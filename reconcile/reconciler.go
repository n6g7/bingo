package reconcile

import (
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/n6g7/bingo/config"
	"github.com/n6g7/bingo/nameserver"
	"github.com/n6g7/bingo/proxy"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	deletionCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bingo_deleted_records",
		Help: "The total number of deleted records",
	})
	creationCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bingo_created_records",
		Help: "The total number of created records",
	})
	managedGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bingo_managed_records",
		Help: "The number of managed records",
	})
)

type Reconciler struct {
	nameserverDomains  *DomainSet
	proxyDomains       *DomainSet
	needsDiff          bool
	proxyBackend       proxy.Proxy
	nsBackend          nameserver.Nameserver
	lastReconciliation time.Time
	minimumWait        time.Duration
	loopTimeout        time.Duration
	deletionQueue      *DomainSet
	conf               *config.Config
}

func NewReconciler(
	ns nameserver.Nameserver,
	prox proxy.Proxy,
	conf *config.Config,
) *Reconciler {
	return &Reconciler{
		nil,
		nil,
		false,
		prox,
		ns,
		time.Unix(0, 0),
		conf.ReconciliationTimeout,
		conf.ReconcilerLoopTimeout,
		NewDomainSet(),
		conf,
	}
}

func (r *Reconciler) SetNameserverDomains(nsDomains *DomainSet) {
	if reflect.DeepEqual(nsDomains, r.nameserverDomains) {
		return
	}
	r.nameserverDomains = nsDomains
	r.needsDiff = true
}

func (r *Reconciler) SetProxyDomains(proxyDomains *DomainSet) {
	if reflect.DeepEqual(proxyDomains, r.proxyDomains) {
		return
	}
	r.proxyDomains = proxyDomains
	r.needsDiff = true
}

func (r *Reconciler) MarkForDeletion(domain string) {
	r.deletionQueue.Add(domain)
	r.needsDiff = true
}

func (r *Reconciler) Diff() (toCreate *DomainSet, toDelete *DomainSet) {
	if r.nameserverDomains == nil {
		log.Println("[DEBUG] Reconciler not ready to diff, no nameserver domains yet")
		return nil, nil
	}
	if r.proxyDomains == nil {
		log.Println("[DEBUG] Reconciler not ready to diff, no proxy domains yet")
		return nil, nil
	}

	toDelete = r.nameserverDomains.Diff(r.proxyDomains).Union(r.deletionQueue)                       // NS - P + D
	toCreate = r.proxyDomains.Diff(r.nameserverDomains).Union(r.deletionQueue.Inter(r.proxyDomains)) // P - NS + (D&P)

	managedGauge.Set(float64(r.proxyDomains.Length()))

	return
}

func (r *Reconciler) Reconcile(toCreate, toDelete *DomainSet) error {
	r.lastReconciliation = time.Now()

	// Start by deleting, gives us a chance to immediately recreate domains in
	// the deletion queue that are in the proxy (they need a new target).
	for domain := range toDelete.Iter() {
		if !r.conf.IsServiceDomain(domain) {
			return fmt.Errorf("Won't delete \"%s\": not a service domain", domain)
		}

		log.Printf("[INFO] Deleting %s...", domain)
		err := r.nsBackend.RemoveRecord(domain)
		if err != nil {
			return fmt.Errorf("Record deletion failed: %w", err)
		} else {
			deletionCounter.Inc()
			log.Println("[DEBUG] Done.")
		}
	}

	for domain := range toCreate.Iter() {
		if !r.conf.IsServiceDomain(domain) {
			return fmt.Errorf("Won't create \"%s\": not a service domain", domain)
		}

		log.Printf("[INFO] Creating %s...", domain)
		target := r.proxyBackend.GetTarget(domain)
		err := r.nsBackend.AddRecord(domain, target)
		if err != nil {
			return fmt.Errorf("Record creation failed: %w", err)
		} else {
			creationCounter.Inc()
			log.Println("[DEBUG] Done.")
		}
	}

	return nil
}

func (r *Reconciler) Run() error {
	tooEarlyWarningSent := false

	for {
		if r.needsDiff {
			toCreate, toDelete := r.Diff()

			if (toCreate == nil || toCreate.Length() == 0) && (toDelete == nil || toDelete.Length() == 0) {
				log.Println("[INFO] Proxy and nameserver are in sync")
				r.needsDiff = false
			} else {
				log.Println("[INFO] Proxy and nameserver are out of sync")

				now := time.Now()
				earliestReco := r.lastReconciliation.Add(r.minimumWait)
				if now.After(earliestReco) {
					log.Println("[DEBUG] Starting reconciliation...")
					err := r.Reconcile(toCreate, toDelete)
					if err != nil {
						log.Printf("[ERROR] Error during reconciliation, will attempt again: %s", err)
					} else {
						r.deletionQueue = NewDomainSet()
						r.needsDiff = false
					}
					tooEarlyWarningSent = false
				} else if !tooEarlyWarningSent {
					log.Printf(
						"[DEBUG] Last reconciliation was less than %s ago, will attempt again in %s\n",
						r.minimumWait,
						earliestReco.Sub(now).Round(time.Second),
					)
					tooEarlyWarningSent = true
				}
			}
		}

		time.Sleep(r.loopTimeout)
	}
}
