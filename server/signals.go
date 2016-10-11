package server

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/uber-go/zap"

	"decipher.com/object-drive-server/services/zookeeper"
)

// TrapSignalsPosix captures POSIX-only signals.
func TrapSignalsPosix(z *zookeeper.ZKState, logger zap.Logger, dp DrainProvider) {
	go func() {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGUSR1)

		for sig := range sigchan {
			switch sig {
			case syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2:
				logger.Info("prepare to die")
				//Stop our zk connection to ensure that we have no more work left
				zookeeper.ServiceStop(z, logger)
				//Wait for our uploaded items to drop to zero
				for {
					//Wait a bit so that we stop getting new work in wait long enough that we are no longer getting new work
					time.Sleep(time.Duration(2) * time.Duration(z.Timeout) * time.Second)
					d, ok := dp.(*S3DrainProviderData)
					if ok {
						//Wait for existing uploads to finish
						if d.CountUploaded() == 0 {
							logger.Info("dying")
							os.Exit(0)
						}
					}
				}
			default:
				os.Exit(1)
			}
		}
	}()
}
