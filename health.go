package diogenes

import (
	"fmt"
	"log"
	"time"
)

func (v *Vault) Health() (bool, error) {
	_, err := v.Connection.Logical().Read("sys/health")
	if err != nil {
		return false, fmt.Errorf("unable to connect to vault: %w", err)
	}

	return true, nil
}

func (v *Vault) CheckHealthyStatus(ticks, tick time.Duration) bool {
	healthy := false

	ticker := time.NewTicker(tick)
	timeout := time.After(ticks)

	for {
		select {
		case t := <-ticker.C:
			log.Printf("tick: %s", t)
			res, err := v.Health()
			if err != nil {
				log.Printf("Error getting response: %s", err)
				continue
			}
			if res {
				healthy = true
				ticker.Stop()
			}

		case <-timeout:
			ticker.Stop()
		}
		break
	}

	return healthy
}
