package reconcile

import (
	"fmt"
	"reflect"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/n6g7/bingo/internal/config"
	"github.com/n6g7/bingo/internal/nameserver"
	"github.com/n6g7/bingo/internal/proxy"
	"github.com/n6g7/nomtail/pkg/log"
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
	logger             *log.Logger
	nameserverDomains  mapset.Set[string]
	proxyDomains       mapset.Set[string]
	needsDiff          bool
	proxyBackend       proxy.Proxy
	nsBackend          nameserver.Nameserver
	lastReconciliation time.Time
	minimumWait        time.Duration
	loopTimeout        time.Duration
	deletionQueue      mapset.Set[string]
	conf               *config.Config
}

func NewReconciler(
	logger *log.Logger,
	ns nameserver.Nameserver,
	prox proxy.Proxy,
	conf *config.Config,
) *Reconciler {
	return &Reconciler{
		logger:             logger.With("component", "reconciler"),
		nameserverDomains:  nil,
		proxyDomains:       nil,
		needsDiff:          false,
		proxyBackend:       prox,
		nsBackend:          ns,
		lastReconciliation: time.Unix(0, 0),
		minimumWait:        conf.ReconciliationTimeout,
		loopTimeout:        conf.ReconcilerLoopTimeout,
		deletionQueue:      mapset.NewSet[string](),
		conf:               conf,
	}
}

func (r *Reconciler) SetNameserverDomains(nsDomains mapset.Set[string]) {
	r.logger.Trace("received NS domains", "domains", nsDomains.ToSlice())
	if reflect.DeepEqual(nsDomains, r.nameserverDomains) {
		return
	}
	r.nameserverDomains = nsDomains
	r.needsDiff = true
}

func (r *Reconciler) SetProxyDomains(proxyDomains mapset.Set[string]) {
	r.logger.Trace("received proxy domains", "domains", proxyDomains.ToSlice())
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

func (r *Reconciler) Diff() (toCreate mapset.Set[string], toDelete mapset.Set[string]) {
	if r.nameserverDomains == nil {
		r.logger.Debug("reconciler not ready to diff, no nameserver domains yet")
		return nil, nil
	}
	if r.proxyDomains == nil {
		r.logger.Debug("reconciler not ready to diff, no proxy domains yet")
		return nil, nil
	}

	toDelete = r.nameserverDomains.Difference(r.proxyDomains).Union(r.deletionQueue)                           // NS - P + D
	toCreate = r.proxyDomains.Difference(r.nameserverDomains).Union(r.deletionQueue.Intersect(r.proxyDomains)) // P - NS + (D&P)

	managedGauge.Set(float64(r.proxyDomains.Cardinality()))

	return
}

func (r *Reconciler) Reconcile(toCreate, toDelete mapset.Set[string]) error {
	r.lastReconciliation = time.Now()

	// Start by deleting, gives us a chance to immediately recreate domains in
	// the deletion queue that are in the proxy (they need a new target).
	for domain := range toDelete.Iter() {
		if !r.conf.IsServiceDomain(domain) {
			return fmt.Errorf("won't delete \"%s\": not a service domain", domain)
		}

		r.logger.Info("deleting domain...", "domain", domain)
		err := r.nsBackend.RemoveRecord(domain)
		if err != nil {
			return fmt.Errorf("record deletion failed: %w", err)
		} else {
			deletionCounter.Inc()
			r.logger.Debug("deleted domain", "domain", domain)
		}
	}

	for domain := range toCreate.Iter() {
		if !r.conf.IsServiceDomain(domain) {
			return fmt.Errorf("won't create \"%s\": not a service domain", domain)
		}

		r.logger.Info("creating domain...", "domain", domain)
		target := r.proxyBackend.GetTarget(domain)
		err := r.nsBackend.AddRecord(domain, target)
		if err != nil {
			return fmt.Errorf("record creation failed: %w", err)
		} else {
			creationCounter.Inc()
			r.logger.Debug("created domain", "domain", domain)
		}
	}

	return nil
}

func (r *Reconciler) Run() error {
	tooEarlyWarningSent := false
	previouslyInSync := false

	for {
		if r.needsDiff {
			toCreate, toDelete := r.Diff()

			if (toCreate == nil || toCreate.Cardinality() == 0) && (toDelete == nil || toDelete.Cardinality() == 0) {
				if !previouslyInSync {
					r.logger.Info("proxy and nameserver are in sync")
					previouslyInSync = true
				}
				r.needsDiff = false
			} else {
				if previouslyInSync {
					r.logger.Info("proxy and nameserver are out of sync")
					previouslyInSync = false
				}

				now := time.Now()
				earliestReco := r.lastReconciliation.Add(r.minimumWait)
				if now.After(earliestReco) {
					r.logger.Debug("starting reconciliation...")
					err := r.Reconcile(toCreate, toDelete)
					if err != nil {
						r.logger.Error("error during reconciliation, will attempt again", "err", err)
					} else {
						r.deletionQueue = mapset.NewSet[string]()
						r.needsDiff = false
					}
					tooEarlyWarningSent = false
				} else if !tooEarlyWarningSent {
					r.logger.Debug(
						"not enough time has passed since the last reconciliation",
						"minimum_wait", r.minimumWait,
						"next_attempt_in", earliestReco.Sub(now).Round(time.Second),
					)
					tooEarlyWarningSent = true
				}
			}
		}

		time.Sleep(r.loopTimeout)
	}
}
