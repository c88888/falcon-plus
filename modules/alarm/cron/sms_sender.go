// Copyright 2017 Xiaomi, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cron

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/open-falcon/falcon-plus/modules/alarm/g"
	"github.com/open-falcon/falcon-plus/modules/alarm/model"
	"github.com/open-falcon/falcon-plus/modules/alarm/redi"
	"github.com/toolkits/net/httplib"
)

func ConsumeSms() {
	for {
		L := redi.PopAllSms()
		if len(L) == 0 {
			time.Sleep(time.Millisecond * 200)
			continue
		}
		SendSmsList(L)
	}
}

func SendSmsList(L []*model.Sms) {
	for _, sms := range L {
		SmsWorkerChan <- 1
		go sendHuyisms(sms)
	}
}

func SendSms(sms *model.Sms) {
	defer func() {
		<-SmsWorkerChan
	}()

	url := g.Config().Api.Sms
	r := httplib.Post(url).SetTimeout(5*time.Second, 30*time.Second)
	r.Param("tos", sms.Tos)
	r.Param("content", sms.Content)
	resp, err := r.String()
	if err != nil {
		log.Errorf("send sms fail, tos:%s, cotent:%s, error:%v", sms.Tos, sms.Content, err)
	}

	log.Debugf("send sms:%v, resp:%v, url:%s", sms, resp, url)
}

//huyi send sms
func sendHuyisms(sms *model.Sms) {
	defer func() {
		<-SmsWorkerChan
	}()
	account := g.Config().HuyiSMS.Account   //APIID
	password := g.Config().HuyiSMS.Password //APIKEY
	mobile, content := sms.Tos, sms.Content //tos,content

	v := url.Values{}
	now := strconv.FormatInt(time.Now().Unix(), 10)

	v.Set("account", account)
	v.Set("password", getMd5String(account+password+mobile+content+now))
	v.Set("mobile", mobile)
	v.Set("content", content)
	v.Set("time", now)

	client := http.DefaultClient

	body := ioutil.NopCloser(strings.NewReader(v.Encode())) //把form数据编下码
	req, err := http.NewRequest("POST", fmt.Sprintf("http://api.isms.ihuyi.com/webservice/isms.php?method=Submit&format=%s", g.Config().HuyiSMS.Format), body)
	if err != nil {
		log.Errorf("cron.sendHuyisms request error:%v", err)
		return
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("cron.sendHuyisms do error:%v", err)
		return
	}

	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("cron.sendHuyisms response error:%v", err)
		return
	}
	log.Infof("cron.sendHuyisms Resp:%v", string(data))
}

func getMd5String(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
