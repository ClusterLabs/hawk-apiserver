package cib

import (
	"fmt"
	pacemaker "github.com/ClusterLabs/go-pacemaker"
	log "github.com/sirupsen/logrus"
	"os"
	"sync"
	"time"
)

// AsyncCib wraps the CIB retrieval from go-pacemaker in an
// asynchronous interface, so that other parts of the server have a
// single copy of the CIB available at any time.
//
// Also provides a subscription interface for the long polling request
// end point, via Wait().
type AsyncCib struct {
	xmldoc   string
	version  *pacemaker.CibVersion
	lock     sync.Mutex
	notifier chan chan string
}

// LogRecord records the last warning and error messages, to avoid
// spamming the log with duplicate messages.
type LogRecord struct {
	warning string
	error   string
}

// Start launches two goroutines, one which runs the go-pacemaker
// mainloop and one which listens for CIB events (the CIB fetcher
// goroutine).
func (acib *AsyncCib) Start() {
	if acib.notifier == nil {
		acib.notifier = make(chan chan string)
	}

	msg := ""
	lastLog := LogRecord{warning: "", error: ""}

	cibFile := os.Getenv("CIB_file")

	cibFetcher := func() {
		for {
			var cib *pacemaker.Cib
			var err error
			if cibFile != "" {
				cib, err = pacemaker.OpenCib(pacemaker.FromFile(cibFile))
			} else {
				cib, err = pacemaker.OpenCib()
			}
			if err != nil {
				msg = fmt.Sprintf("Failed to connect to Pacemaker: %v", err)
				if msg != lastLog.warning {
					log.Warnf(msg)
					lastLog.warning = msg
				}
				time.Sleep(5 * time.Second)
			}
			for cib != nil {
				func() {
					cibxml, err := cib.Query()
					if err != nil {
						msg = fmt.Sprintf("Failed to query CIB: %v", err)
						if msg != lastLog.error {
							log.Errorf(msg)
							lastLog.error = msg
						}
					} else {
						acib.notifyNewCib(cibxml)
					}
				}()

				waiter := make(chan int)
				_, err = cib.Subscribe(func(event pacemaker.CibEvent, doc *pacemaker.CibDocument) {
					if event == pacemaker.UpdateEvent {
						acib.notifyNewCib(doc)
					} else {
						msg = fmt.Sprintf("lost connection: %v", event)
						if msg != lastLog.warning {
							log.Warnf(msg)
							lastLog.warning = msg
						}
						waiter <- 1
					}
				})
				if err != nil {
					log.Infof("Failed to subscribe, rechecking every 5 seconds")
					time.Sleep(5 * time.Second)
				} else {
					<-waiter
				}
			}
		}
	}

	go cibFetcher()
	go pacemaker.Mainloop()
}

// Wait blocks for up to `timeout` seconds for a CIB change event.
func (acib *AsyncCib) Wait(timeout int, defval string) string {
	requestChan := make(chan string)
	select {
	case acib.notifier <- requestChan:
	case <-time.After(time.Duration(timeout) * time.Second):
		return defval
	}
	return <-requestChan
}

// Get returns the current CIB XML document (or nil).
func (acib *AsyncCib) Get() string {
	acib.lock.Lock()
	defer acib.lock.Unlock()
	return acib.xmldoc
}

// Version returns the current CIB version (or nil).
func (acib *AsyncCib) Version() *pacemaker.CibVersion {
	acib.lock.Lock()
	defer acib.lock.Unlock()
	return acib.version
}

func (acib *AsyncCib) notifyNewCib(cibxml *pacemaker.CibDocument) {
	text := cibxml.ToString()
	version := cibxml.Version()
	log.Infof("[CIB]: %v", version)
	acib.lock.Lock()
	acib.xmldoc = text
	acib.version = version
	acib.lock.Unlock()
	// Notify anyone waiting
Loop:
	for {
		select {
		case clientchan := <-acib.notifier:
			clientchan <- version.String()
		default:
			break Loop
		}
	}
}
