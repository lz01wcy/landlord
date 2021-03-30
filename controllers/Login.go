package controllers

import (
	"crypto/md5"
	"fmt"
	"github.com/astaxie/beego/logs"
	"landlord/common"
	"net/http"
	"strconv"
)

//登录
func Login(w http.ResponseWriter, r *http.Request) {
	// defer处理panic
	defer func() {
		if r := recover(); r != nil {
			logs.Error("user request Login - Login panic:%v ", r)
		}
	}()
	var ret = []byte{'1'}
	// 第二个 defer处理
	defer func() {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		_, err := w.Write(ret)
		if err != nil {
			logs.Error("")
		}
	}()
	// 读取email
	username := r.FormValue("username")
	// 如果长度为0 从PostFormValue中读取
	if len(username) == 0 {
		username = r.PostFormValue("username")
		// 还是长度为0 error
		if username == "" {
			logs.Error("user request Login - err: username is empty")
			return
		}
	}
	// 读取email
	email := r.FormValue("email")
	// 如果长度为0 从PostFormValue中读取
	if len(email) == 0 {
		email = r.PostFormValue("email")
		// 还是长度为0 error
		if email == "" {
			logs.Error("user request Login - err: email is empty")
			return
		}
	}
	// 取得密码 明文的?
	password := r.FormValue("password")
	if len(password) == 0 {
		password = r.PostFormValue("password")
		if password == "" {
			logs.Error("user request Login - err: password is empty")
			return
		}
	}
	// 密码的MD5
	md5Password := fmt.Sprintf("%x", md5.Sum([]byte(password)))
	// 新建账号结构体
	var account = common.Account{}
	//err := common.GameConfInfo.MysqlConf.Pool.Get(&account, "select * from account where username=? and password", email,md5Password)
	// 读数据库,email默认为用户名?
	row := common.GameConfInfo.Db.QueryRow("select * from account where username=? and password=?", username, md5Password)
	if row != nil {
		// 转为账号结构体数值
		err := row.Scan(&account.Id, &account.Email, &account.Username, &account.Password, &account.Coin, &account.CreatedDate, &account.UpdateDate)
		if err != nil {
			// 写入cookie
			cookie := http.Cookie{Name: "userid", Value: strconv.Itoa(account.Id), Path: "/", MaxAge: 86400}
			http.SetCookie(w, &cookie)
			cookie = http.Cookie{Name: "username", Value: account.Username, Path: "/", MaxAge: 86400}
			http.SetCookie(w, &cookie)
		}
	}
}
