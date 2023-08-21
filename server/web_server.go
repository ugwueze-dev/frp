package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/fatedier/frp/pkg/config"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/util/log"
)

func (svr *Service) resetNicHandler(w http.ResponseWriter, r *http.Request) {
	remotePort := r.URL.Path
	ctls := svr.ctlManager.GetAll()

outer:
	for _, ctl := range ctls {
		proxies := ctl.proxies
		for _, pxy := range proxies {
			t := pxy.GetConf().GetBaseConfig().ProxyType
			if t == "tcp" || t == "udp" {
				pxyPort := pxy.GetConf().(*config.TCPProxyConf).RemotePort
				if strconv.Itoa(pxyPort) == remotePort {

					ctl.sendCh <- &msg.ResetNIC{
						Port: pxyPort,
					}
					log.Info("sent command to reset NIC with remote port %d", pxyPort)
					break outer
				}
			}
		}
	}
}

func (svr *Service) startNicControlServer(addr string, port int64) {
	if addr == "" {
		return
	}

	log.Info("starting reset nic server on %s", fmt.Sprintf("%s:%d", addr, port))
	mux := http.NewServeMux()
	changeIPPath := "/changeip/"
	mux.Handle(changeIPPath, http.StripPrefix(changeIPPath, http.HandlerFunc(svr.resetNicHandler)))

	listenAddr := fmt.Sprintf("%s:%d", addr, port)
	http.ListenAndServe(listenAddr, mux)
}
