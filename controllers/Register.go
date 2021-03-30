package controllers

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego/logs"
	"landlord/common"
	"net/http"
	"strconv"
	"time"
)

// 注册
func Register(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			logs.Error("Register panic:%v ", r)
		}
	}()
	var ret = []byte{'1'}
	defer func() {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		_, err := w.Write(ret)
		if err != nil {
			logs.Error("user request Register - err : %v", err)
		}
	}()
	// 读取用户名和密码
	username := r.FormValue("username")
	if len(username) == 0 {
		username = r.PostFormValue("username")
		if username == "" {
			logs.Error("register err: username is empty")
			return
		}
	}
	password := r.FormValue("password")
	if len(password) == 0 {
		password = r.PostFormValue("password")
		if password == "" {
			logs.Error("register err: password is empty")
			return
		}
	}
	logs.Debug("user request register : username=%v, password=%v ", username, password)

	// 账号结构体
	var account = common.Account{}
	// 读数据库
	row := common.GameConfInfo.Db.QueryRow("select * from account where username=?", username)
	// 写入账户信息
	err := row.Scan(&account.Id, &account.Email, &account.Username, &account.Password, &account.Coin, &account.CreatedDate, &account.UpdateDate)
	// err不空 即没读取 即未注册
	if err != nil {
		now := time.Now().Format("2006-01-02 15:04:05")
		md5Password := fmt.Sprintf("%x", md5.Sum([]byte(password)))
		stmt, err := common.GameConfInfo.Db.Prepare(`insert into account(email,username,password,coin,created_date,updated_date) values(?,?,?,?,?,?) `)
		if err != nil {
			logs.Error("insert new account [%v] err : %v", username, err)
			return
		}
		result, err := stmt.Exec(username, username, md5Password, 10000, now, now)
		if err != nil {
			logs.Error("insert new account [%v] err : %v", username, err)
			return
		}
		lastInsertId, err := result.LastInsertId()
		if err != nil {
			logs.Error("insert new account [%v] get last insert id err : %v", username, err)
			return
		}
		ret, err = json.Marshal(map[string]interface{}{"uid": lastInsertId, "username": username})
		if err != nil {
			logs.Error("json marsha1 failed err:%v", err)
			return
		}
		cookie := http.Cookie{Name: "userid", Value: strconv.Itoa(int(lastInsertId)), Path: "/", MaxAge: 86400}
		http.SetCookie(w, &cookie)
		cookie = http.Cookie{Name: "username", Value: username, Path: "/", MaxAge: 86400}
		http.SetCookie(w, &cookie)
	} else {
		logs.Debug("user [%v] request register err: user already exists", username)
	}
}
