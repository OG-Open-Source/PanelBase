package api

import (
	"net/http"
	"github.com/gorilla/mux"
)

func SetupRoutes() *mux.Router {
	router := mux.NewRouter()

	// 安全入口路由
	router.HandleFunc("/{securityEntry}/status", statusHandler).Methods("GET")
	router.HandleFunc("/{securityEntry}/command", commandHandler).Methods("POST")

	return router
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("PanelBase agent 正在運行"))
}

func commandHandler(w http.ResponseWriter, r *http.Request) {
	// 處理來自 panel.ogtt.tk 的命令
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("命令已接收"))
}