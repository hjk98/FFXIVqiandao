package main

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

var (
	header = map[string]string{
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3",
		"Accept-Encoding":           "gzip, deflate, br",
		"Accept-Language":           "zh-CN,zh;q=0.9,en;q=0.8",
		"Cache-Control":             "no-cache",
		"Connection":                "keep-alive",
		"Cookie":                    "CASCID=CID2BFCADB9A0154A9D877294366E144906; sdo_cas_id=10.129.20.137; CAS_LOGIN_STATE=1; sdo_dw_track=G81Y/L1voXjLY8VH5ZWfpw==; CASTGC=ULSTGT-f0caef48519646a09e4ecee2f864e40a",
		"Host":                      "cas.sdo.com",
		"Pragma":                    "no-cache",
		"Sec-Fetch-Mode":            "navigate",
		"Sec-Fetch-Site":            "none",
		"Sec-Fetch-User":            "?1",
		"Upgrade-Insecure-Requests": "1",
		"User-Agent":                "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/76.0.3809.100 Safari/537.36",
	}
	m = map[string]string{
		"陆行鸟": "1",
		"莫古力": "6",
		"猫小胖": "7",
	}
)

func main() {
	c := cron.New()
	accounts := getAccounts()
	for _, acc := range accounts {
		_, err := c.AddJob("@daily", acc)
		if err != nil {
			log.Fatalf("Fail to AddJob: %v\n", err)
		}
	}
	c.Start()
	defer c.Stop()
	select {}
}

func (acc Account) Run() {
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatalf("Fail to New: %v\n", err)
	}
	c := &http.Client{
		Jar: jar,
	}

	tic := step1(acc, c)
	step2(acc, c)
	step3(c)
	step4(tic, c)
	role := step5(acc, c)
	step6(acc, role, c)
	step7(c)
	step8(c)
}

func setheader(h map[string]string, req *http.Request) {
	for k, v := range h {
		req.Header.Set(k, v)
	}
}

func getURL(h string, p map[string]string) string {
	u, err := url.Parse(h)
	if err != nil {
		log.Fatalf("Fail to Parse: %v\n", err)
	}
	q := u.Query()
	for k, v := range p {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// 提交用户名和密码，获取ticket
func step1(acc Account, client *http.Client) string {
	host := "https://cas.sdo.com/authen/staticLogin.jsonp"
	params := map[string]string{
		"callback":            "staticLogin_JSONPMethod",
		"inputUserId":         acc.UserId,
		"password":            acc.Password,
		"appId":               "100001900",
		"areaId":              "1",
		"serviceUrl":          "http://act.ff.sdo.com/20180707jifen/Server/SDOLogin.ashx?returnPage=index.html",
		"productVersion":      "v5",
		"frameType":           "3",
		"locale":              "zh_CN",
		"version":             "21",
		"tag":                 "20",
		"authenSource":        "2",
		"productId":           "2",
		"scene":               "login",
		"usage":               "aliCode",
		"customSecurityLevel": "2",
		"autoLoginFlag":       "0",
		"_":                   strconv.Itoa(int(time.Now().Unix() * 1000)),
	}

	u := getURL(host, params)

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		log.Fatalf("Fail to NewRequest: %v\n", err)
	}
	setheader(header, req)

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Fail to Do: %v\n", err)
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Fatalf("Fail to Close: %v\n", err)
		}
	}()

	var reader io.ReadCloser
	if resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			log.Fatalf("Fail to NewReader: %v\n", err)
		}
	} else {
		reader = resp.Body
	}
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Fatalf("Fail to ReadAll: %v\n", err)
	}

	text := string(b)
	text = text[strings.Index(text, "(")+1 : strings.LastIndex(text, ")")]
	respBody := &struct {
		ReturnCode    int    `json:"return_code"`
		ErrorType     int    `json:"error_type"`
		ReturnMessage string `json:"return_message"`
		Data          struct {
			AppID          int    `json:"appId"`
			AreaID         int    `json:"areaId"`
			IsNeedFullInfo int    `json:"isNeedFullInfo"`
			NextAction     int    `json:"nextAction"`
			SndaID         string `json:"sndaId"`
			Ticket         string `json:"ticket"`
		} `json:"data"`
	}{}

	err = json.Unmarshal([]byte(text), respBody)
	if err != nil {
		log.Fatalf("Fail to Unmarshal: %v\n", err)
	}

	ticket := respBody.Data.Ticket
	if len(ticket) == 0 {
		log.Fatalln(acc.RoleName, "登录失败，短期内登录失败次数过多，服务器已开启验证码，请在1-3天后再试...")
	} else {
		log.Println(acc.RoleName, "登录成功，正在设置cookie...")
	}
	return ticket
}

// 设置Cookie
func step2(acc Account, client *http.Client) {
	host := "http://login.sdo.com/sdo/Login/Tool.php"
	params := map[string]string{
		"value": "index|" + acc.UserId,
		"act":   "setCookie",
		"name":  "CURRENT_TAB",
		"r":     "0.8326684884385089",
	}
	u := getURL(host, params)
	_, err := client.Get(u)
	if err != nil {
		log.Fatalf("Fail to Get: %v\n", err)
	}
}

// 设置Cookie
func step3(client *http.Client) {
	host := "https://cas.sdo.com/authen/getPromotionInfo.jsonp"
	params := map[string]string{
		"callback":            "getPromotionInfo_JSONPMethod",
		"appId":               "991000350",
		"areaId":              "1001",
		"serviceUrl":          "http://act.ff.sdo.com/20180707jifen/Server/SDOLogin.ashx?returnPage=index.html",
		"productVersion":      "v5",
		"frameType":           "3",
		"locale":              "zh_CN",
		"version":             "21",
		"tag":                 "20",
		"authenSource":        "2",
		"productId":           "2",
		"scene":               "login",
		"usage":               "aliCode",
		"customSecurityLevel": "2",
		"_":                   "1566623599098",
	}
	u := getURL(host, params)
	_, err := client.Get(u)
	if err != nil {
		log.Fatalf("Fail to Get: %v\n", err)
	}
}

// 设置Cookie
func step4(ticket string, client *http.Client) {
	u := "http://act.ff.sdo.com/20180707jifen/Server/SDOLogin.ashx?returnPage=index.html&ticket=" + ticket
	_, err := client.Get(u)
	if err != nil {
		log.Fatalf("Fail to Get: %v\n", err)
	}
}

// 查询角色列表
func step5(acc Account, client *http.Client) string {
	host := "http://act.ff.sdo.com/20180707jifen/Server/ff14/HGetRoleList.ashx"
	ipid, ok := m[acc.AreaName]
	if !ok {
		log.Fatalln("大区名称有误")
	}
	params := map[string]string{
		"method": "queryff14rolelist",
		"ipid":   ipid,
		"i":      "0.8075943537407986",
	}

	u := getURL(host, params)
	resp, err := client.Get(u)
	if err != nil {
		log.Fatalf("Fail to Get: %v\n", err)
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Fatalf("Fail to Close: %v\n", err)
		}
	}()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Fail to ReadAll: %v\n", err)
	}
	respBody := &struct {
		Code    int    `json:"Code"`
		Message string `json:"Message"`
		Attach  []struct {
			Cicuid          string      `json:"cicuid"`
			AreaName        interface{} `json:"areaName"`
			GroupName       interface{} `json:"groupName"`
			RealRoleName    interface{} `json:"realRoleName"`
			Name            string      `json:"name"`
			Worldname       string      `json:"worldname"`
			Characterstatus int         `json:"characterstatus"`
			Lodestoneid     interface{} `json:"lodestoneid"`
			Renameflag      bool        `json:"renameflag"`
			WorldnameZh     string      `json:"worldnameZh"`
			Ipid            int         `json:"ipid"`
			Groupid         int         `json:"groupid"`
			AreaID          int         `json:"AreaId"`
			Characterid     interface{} `json:"characterid"`
		} `json:"Attach"`
		Success bool `json:"Success"`
	}{}
	err = json.Unmarshal(b, respBody)
	if err != nil {
		log.Fatalf("Fail to Unmarshal: %v\n", err)
	}

	log.Println("正在获取角色列表...")
	for _, r := range respBody.Attach {
		if r.WorldnameZh == acc.ServerName && r.Name == acc.RoleName {
			log.Println("获取角色列表成功...")
			role := strings.Join([]string{r.Cicuid, r.Worldname, strconv.Itoa(r.Groupid)}, "|")
			return role
		}
	}
	log.Fatalln("获取角色列表失败...")
	return ""
}

// 选择区服及角色
func step6(acc Account, role string, client *http.Client) {
	host := "http://act.ff.sdo.com/20180707jifen/Server/ff14/HGetRoleList.ashx"
	areaId, ok := m[acc.AreaName]
	if !ok {
		log.Fatalln("大区名称有误")
	}
	params := map[string]string{
		"method":   "setff14role",
		"AreaId":   areaId,
		"AreaName": acc.AreaName,
		"RoleName": "[" + acc.ServerName + "]" + acc.RoleName,
		"Role":     role,
		"i":        "0.8326684884385089",
	}

	p := url.Values{}
	for k, v := range params {
		p.Set(k, v)
	}
	_, err := client.Post(host, "application/x-www-form-urlencoded", strings.NewReader(p.Encode()))
	if err != nil {
		log.Fatalf("Fail to Post: %v\n", err)
	}
	log.Println("已选择目标角色...")
}

// 签到
func step7(client *http.Client) {
	host := "http://act.ff.sdo.com/20180707jifen/Server/User.ashx"
	log.Println("正在签到...")
	params := map[string]string{
		"method": "signin",
		"i":      "0.855755357775076",
	}

	p := url.Values{}
	for k, v := range params {
		p.Set(k, v)
	}
	resp, err := client.Post(host, "application/x-www-form-urlencoded", strings.NewReader(p.Encode()))
	if err != nil {
		log.Fatalf("Fail to Post: %v\n", err)
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Fatalf("Fail to Close: %v\n", err)
		}
	}()

	rb, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Fail to ReadAll: %v\n", err)
	}
	respBody := &struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Attach  string `json:"Attach"`
		Success bool   `json:"success"`
	}{}
	err = json.Unmarshal(rb, respBody)
	if err != nil {
		log.Fatalf("Fail to Unmarshal: %v\n", err)
	}
	log.Println(respBody.Message)
}

// 查询当前积分
func step8(client *http.Client) {
	host := "http://act.ff.sdo.com/20180707jifen/Server/User.ashx"
	params := map[string]string{
		"method": "querymystatus",
		"i":      "0.855755357775076",
	}

	p := url.Values{}
	for k, v := range params {
		p.Set(k, v)
	}
	resp, err := client.Post(host, "application/x-www-form-urlencoded", strings.NewReader(p.Encode()))
	if err != nil {
		log.Fatalf("Fail to Post: %v\n", err)
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Fatalf("Fail to Close: %v\n", err)
		}
	}()

	rb, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Fail to ReadAll: %v\n", err)
	}
	respBody := &struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Attach  string `json:"Attach"`
		Success bool   `json:"success"`
	}{}
	err = json.Unmarshal(rb, respBody)
	if err != nil {
		log.Fatalf("Fail to Unmarshal: %v\n", err)
	}
	attach := &struct {
		Jifen     int    `json:"Jifen"`
		PtAccount string `json:"ptAccount"`
	}{}
	err = json.Unmarshal([]byte(respBody.Attach), attach)
	if err != nil {
		log.Fatalf("Fail to Unmarshal: %v\n", err)
	}
	log.Printf("当前积分为：%d\n", attach.Jifen)
}
