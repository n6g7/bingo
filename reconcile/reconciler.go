package reconcile

import (
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/n6g7/bingo/nameserver"
	"github.com/n6g7/bingo/proxy"
)

type Reconciler struct {
	nameserverDomains  *DomainSet
	proxyDomains       *DomainSet
	needsDiff          bool
	proxyBackend       proxy.Proxy
	nsBackend          nameserver.Nameserver
	lastReconciliation time.Time
	minimumWait        time.Duration
}

func NewReconciler(
	ns nameserver.Nameserver,
	prox proxy.Proxy,
	minimumWait time.Duration,
) *Reconciler {
	return &Reconciler{
		nil,
		nil,
		false,
		prox,
		ns,
		time.Unix(0, 0),
		minimumWait,
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

func (r *Reconciler) Diff() (*DomainSet, *DomainSet) {
	if r.nameserverDomains == nil {
		log.Println("[DEBUG] Reconciler not ready to diff, no nameserver domains yet")
		return nil, nil
	}
	if r.proxyDomains == nil {
		log.Println("[DEBUG] Reconciler not ready to diff, no proxy domains yet")
		return nil, nil
	}

	toCreate := r.proxyDomains.Diff(r.nameserverDomains)
	toDelete := r.nameserverDomains.Diff(r.proxyDomains)

	return toCreate, toDelete
}

func (r *Reconciler) Reconcile(toCreate, toDelete *DomainSet) error {
	r.lastReconciliation = time.Now()

	for domain := range toCreate.Iter() {
		log.Printf("[INFO] Creating %s...", domain)
		target := r.proxyBackend.GetTarget(domain)
		err := r.nsBackend.AddRecord(domain, target)
		if err != nil {
			return fmt.Errorf("Record creation failed: %w", err)
		} else {
			log.Println("[DEBUG] Done.")
		}
	}
	for domain := range toDelete.Iter() {
		log.Printf("[INFO] Deleting %s...", domain)
		err := r.nsBackend.RemoveRecord(domain)
		if err != nil {
			return fmt.Errorf("Record deletion failed: %w", err)
		} else {
			log.Println("[DEBUG] Done.")
		}
	}

	return nil
}

func (r *Reconciler) Run() error {
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
						r.needsDiff = false
					}
				} else {
					log.Printf(
						"[DEBUG] Last reconciliation was less than %s ago, will attempt again in %s\n",
						r.minimumWait,
						earliestReco.Sub(now).Round(time.Second),
					)
				}
			}
		}

		time.Sleep(500 * time.Millisecond)
	}
}
