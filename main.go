package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/aliyun/fc-runtime-go-sdk/fc"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

// 阿里云 账号及OSS设置

const AliyunAccess = "LTAI5tFPDJNHMdBvn6dSPEC8"
const AliyunAccessSecret = "U1LYgMNuZC8uACemDDDrPxnbnViNmC"
const OSSEndpoint = "https://oss-cn-hangzhou.aliyuncs.com"
const OSSBucket = "sentry-white-api"

// ApiPrefix api前缀
const ApiPrefix = "api/v1/"

// DingTalkUrl 钉钉消息地址
const DingTalkUrl = "https://oapi.dingtalk.com/robot/send?access_token="

type Project struct {
	ID                  uint64 `json:"id"`
	Name                string `json:"name"`
	SubstituteName      string `json:"substitute_name"`
	Robot               `json:"robot"`
	ResponsiblePeopleID []uint64 `json:"responsible_people_id"`
	OriginalID          uint64   `json:"original_id"`
	DeletedAt           string   `json:"deleted_at"`
}
type ProjectsCollection struct {
	Projects []Project `json:"projects"`
}

type Robot struct {
	ID          uint64 `json:"id"`
	AccessToken string `json:"access_token"`
}

type ResponsiblePerson struct {
	ID    uint64 `json:"id"`
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

type Original struct {
	AccountName string `json:"account_name"`
	Password    string `json:"password"`
}

type Message struct {
	ProjectSlug string            `json:"project_slug,omitempty"`
	Url         string            `json:"url,omitempty"`
	Event       map[string]string `json:"event,omitempty"`
}

func main() {
	fc.StartHttp(HandleHttpRequest)
}

// HandleHttpRequest ...
func HandleHttpRequest(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
	bodyByte, err := ioutil.ReadAll(req.Body)
	logIf(err)
	var message Message
	err = json.Unmarshal(bodyByte, &message)

	fmt.Println(message)

	if message.ProjectSlug == "" {
		if bucketConnectionSuccess() {
			_, err := fmt.Fprintf(w, "部署成功，请按手册指引进行后续步骤")
			logIf(err)
		} else {
			_, err := fmt.Fprintf(w, "OSS 连接失败，请检查参数设置是否正确")
			logIf(err)
		}
		return nil
	}

	// 获取项目列表
	project := Project{ID: 0}
	projects := getProjects()

	for _, p := range projects {
		fmt.Println(p)
		if p.Name == message.ProjectSlug && p.DeletedAt == "0001-01-01 00:00:00" {
			project = p
		}
	}
	if project.ID == 0 {
		_, err := fmt.Fprintf(w, "未找到匹配项目，消息终止")
		logIf(err)
		fmt.Println("未找到匹配项目，消息终止")
		return nil
	}

	fmt.Println("最终项目信息")
	fmt.Println(project)
	// 获取详细全套信息
	project = getProject(project.ID)
	robot := getRobot(project.Robot.ID)
	rps := getResponsiblePeople(project.ResponsiblePeopleID)
	original := getOriginal(project.OriginalID)

	fmt.Println(original)
	fmt.Println(project)
	fmt.Println(robot)
	fmt.Println(rps)

	// 发送post信息
	for _, rp := range rps {
		sendDingTalk(robot.AccessToken, message.Event["title"], message.Url, original.AccountName, original.Password, rp.Phone)
	}

	_, err = fmt.Fprintf(w, "success")
	logIf(err)
	return nil
}

func sendDingTalk(accessToken string, title string, url string, accountName string, password string, phone string) {
	var markdown map[string]string
	markdown = make(map[string]string)
	markdown["title"] = title
	markdown["text"] = fmt.Sprintf("## %s \n"+
		"* 详情地址：%s \n"+
		"* 账号：%s \n"+
		"* 密码：%s \n"+
		"* @%s", title, url, accountName, password, phone)

	var atMobiles map[string][]string
	atMobiles = make(map[string][]string)
	atMobiles["atMobiles"] = []string{phone}

	var message map[string]interface{}
	message = make(map[string]interface{})
	message["msgtype"] = "markdown"
	message["markdown"] = markdown
	message["at"] = atMobiles

	post(DingTalkUrl+accessToken, message)

}

func post(targetUrl string, message interface{}) *http.Response {
	s, err := json.Marshal(&message)
	logIf(err)

	payload := strings.NewReader(string(s))

	fmt.Println(targetUrl)
	fmt.Println(payload)

	req, _ := http.NewRequest("POST", targetUrl, payload)
	req.Header.Add("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(req)
	logIf(err)

	fmt.Println(response.Body)

	return response
}

func getProjects() []Project {
	wanted := ProjectsCollection{}
	response := getFromOSS(ApiPrefix + "projects")
	err := json.Unmarshal([]byte(response), &wanted)
	logIf(err)
	return wanted.Projects
}
func getProject(id uint64) Project {
	project := Project{}
	response := getFromOSS(ApiPrefix + "projects/" + strconv.FormatUint(id, 10))
	err := json.Unmarshal([]byte(response), &project)
	logIf(err)
	return project
}
func getRobot(id uint64) Robot {
	robot := Robot{}
	response := getFromOSS(ApiPrefix + "robots/" + strconv.FormatUint(id, 10))
	err := json.Unmarshal([]byte(response), &robot)
	logIf(err)
	return robot
}
func getOriginal(id uint64) Original {
	original := Original{}
	response := getFromOSS(ApiPrefix + "originals/" + strconv.FormatUint(id, 10))
	err := json.Unmarshal([]byte(response), &original)
	logIf(err)
	return original
}
func getResponsiblePeople(ids []uint64) []ResponsiblePerson {
	var rpl []ResponsiblePerson
	for _, d := range ids {
		rp := ResponsiblePerson{}
		response := getFromOSS(ApiPrefix + "responsible_people/" + strconv.FormatUint(d, 10))
		err := json.Unmarshal([]byte(response), &rp)
		logIf(err)
		rpl = append(rpl, rp)
	}
	return rpl
}

// getFromOSS 发送GET请求
// fileName：         文件名
// response：    请求返回的内容
func getFromOSS(fileName string) string {
	bucket := getBucket()
	signedURL, err := bucket.SignURL(fileName, oss.HTTPGet, 600)
	logIf(err)

	response, _ := http.Get(signedURL)
	// response.Body类型为io.ReadCloser
	//fmt.Printf(response.Body)

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(response.Body)
	if err != nil {
		return ""
	}

	return buf.String()
}

// bucketConnectionSuccess 判断OSS权限是否通过
func bucketConnectionSuccess() (success bool) {
	c, err := oss.New(
		OSSEndpoint,
		AliyunAccess,
		AliyunAccessSecret,
		oss.Timeout(10, 120),
	)
	logIf(err)
	stat, err := c.GetBucketInfo(OSSBucket)
	if err != nil || stat.BucketInfo.Name == "" {
		return false
	}
	return true
}

// 获取 bucket 对象
func getBucket() (b *oss.Bucket) {
	c, err := oss.New(
		OSSEndpoint,
		AliyunAccess,
		AliyunAccessSecret,
		oss.Timeout(10, 120),
	)
	logIf(err)
	b, err = c.Bucket(OSSBucket)
	logIf(err)
	return
}

// logIf 控制台输出错误信息
func logIf(err error) {
	if err != nil {
		fmt.Println(err)
	}
}
