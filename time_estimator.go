package insight

import (
	"sync/atomic"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/dynamicgo/config"
	"github.com/inwecrypto/ethgo/rpc"
)

// BlockTimeEstimator .
type BlockTimeEstimator struct {
	blkPerSecond atomic.Value
	c            *rpc.Client
}

func newBlockTimeEstimator(conf *config.Config) *BlockTimeEstimator {

	est := &BlockTimeEstimator{}
	est.c = rpc.NewClient(conf.GetString("insight.geth", "http://localhost:8545"))

	est.blkPerSecond.Store(float64(0.1))

	go est.estimateTask()

	return est
}

func (est *BlockTimeEstimator) getBPS() float64 {
	return est.blkPerSecond.Load().(float64)
}

func (est *BlockTimeEstimator) estimateTask() {
	ticker := time.NewTicker(time.Second * 10)
	var lastMesaure time.Time
	var lastblkID uint64

	for {
		if blkID, err := est.c.BlockNumber(); err == nil {
			lastblkID = blkID
			lastMesaure = time.Now()
			break
		}
	}

	for _ = range ticker.C {
		if blkID, err := est.c.BlockNumber(); err == nil {
			if blkID > lastblkID {
				bps := float64(blkID-lastblkID) / time.Now().Sub(lastMesaure).Seconds()
				est.blkPerSecond.Store(bps)
				lastblkID = blkID
				lastMesaure = time.Now()
				log.Println("blk/sec:", est.blkPerSecond.Load())
			}
		} else {
			log.Println(err)
		}
	}
}
