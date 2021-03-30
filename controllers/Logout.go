package controllers

import (
	"github.com/astaxie/beego/logs"
	"net/http"
)

// 登出
func LoginOut(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			logs.Error("LoginOut panic:%v ", r)
		}
	}()
	// 清除cookie
	cookie := http.Cookie{Name: "user", Path: "/", MaxAge: -1}
	http.SetCookie(w, &cookie)
	// 这是啥?
	_, err := w.Write([]byte{'1'})
	if err != nil {
		logs.Error("LoginOut err: %v", err)
	}
}
