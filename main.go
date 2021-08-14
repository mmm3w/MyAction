package main

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

var actID string = "e202009291139501"
var region string = "cn_gf01"
var appVer string
var uid string
var cookie string
var userAgent string
var referer string

type (
	Result struct {
		Code int             `json:"retcode"`
		Msg  string          `json:"message"`
		Data json.RawMessage `json:"data"`
	}

	Info struct {
		TotalSignDay int    `json:"total_sign_day"`
		Today        string `json:"today"`
		IsSign       bool   `json:"is_sign"`
		FirstBind    bool   `json:"first_bind"`
		IsSub        bool   `json:"is_sub"`
		MonthFirst   bool   `json:"month_first"`
	}

	SignInfo struct {
		Act    string `json:"act_id"`
		Region string `json:"region"`
		Uid    string `json:"uid"`
	}
)

func (r *Result) parseData(data interface{}) error {
	return json.Unmarshal([]byte(r.Data), &data)
}

func RandomCode() string {
	letterBytes := "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}

func AttachReqInfo(request *http.Request) {
	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("Referer", referer)
	request.Header.Set("Accept-Encoding", "gzip, deflate, br")
	request.Header.Set("Cookie", cookie)

	iiid := uuid.NewV3(uuid.NamespaceURL, cookie)
	request.Header.Set("x-rpc-device_id", strings.ToUpper(strings.Replace(iiid.String(), "-", "", -1)))
	request.Header.Set("x-rpc-client_type", "5")
	request.Header.Set("x-rpc-app_version", appVer)
	t := time.Now().Unix()
	c := RandomCode()
	m := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("salt=h8w582wxwgqvahcdkpvdhbh2w9casgfl&t=%v&r=%v", t, c))))
	request.Header.Set("DS", fmt.Sprintf("%v,%v,%v", t, c, m))
}

func GetInfo(info *Info) error {
	infoUrl := fmt.Sprintf("https://api-takumi.mihoyo.com/event/bbs_sign_reward/info?region=%v&act_id=%v&uid=%v", region, actID, uid)
	infoReq, _ := http.NewRequest("GET", infoUrl, nil)
	AttachReqInfo(infoReq)
	infoResp, err := (&http.Client{}).Do(infoReq)
	if err != nil {
		return err
	}
	defer infoResp.Body.Close()
	respByte, _ := ioutil.ReadAll(infoResp.Body)

	var result Result
	if err := json.Unmarshal(respByte, &result); err != nil {
		return err
	}
	if result.Code != 0 {
		return errors.New(result.Msg)
	}
	if err := result.parseData(info); err != nil {
		return err
	}
	return nil
}

func TrySign() error {
	var params = map[string]string{"act_id": actID, "region": region, "uid": uid}
	var paramJson, _ = json.Marshal(params)
	signReq, _ := http.NewRequest("POST", "https://api-takumi.mihoyo.com/event/bbs_sign_reward/sign", bytes.NewBuffer(paramJson))
	AttachReqInfo(signReq)
	signResp, err := (&http.Client{}).Do(signReq)
	if err != nil {
		return err
	}
	defer signResp.Body.Close()
	respByte, _ := ioutil.ReadAll(signResp.Body)

	var result Result
	if err := json.Unmarshal(respByte, &result); err != nil {
		return err
	}

	if result.Code != 0 {
		return errors.New(result.Msg)
	}

	fmt.Println(string(respByte))
	return nil
}

func main() {
	appVer = os.Args[1]
	uid = os.Args[2]
	cookie = os.Args[3]

	userAgent = fmt.Sprintf("Mozilla/5.0 (iPhone; CPU iPhone OS 14_0_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) miHoYoBBS/%v", appVer)
	referer = fmt.Sprintf("https://webstatic.mihoyo.com/bbs/event/signin-ys/index.html?bbs_auth_required=%v&act_id=%v&utm_source=%v&utm_medium=%v&utm_campaign=%v",
		"true", actID, "bbs", "mys", "icon")

	var info Info

	if err := GetInfo(&info); err != nil {
		fmt.Println(err)
	}

	if !info.IsSign {
		TrySign()
	} else {
		fmt.Println("已签到")
	}
}
