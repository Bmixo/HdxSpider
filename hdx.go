package main

import (
	"bytes"
	"crypto/des"
	"crypto/md5"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/V-I-C-T-O-R/gorc"
	"github.com/buger/jsonparser"
)

type HDX struct {
	userName   string
	passWord   string
	Client     http.Client
	Cookies    []*http.Cookie
	courses    []Course
	target     []int
	courseData map[string][]class
}

type Course struct {
	name     string
	sesson   string
	courseId string
}

type sub struct {
	name string
	url  string
}

type class struct {
	path []string
	name string
	url  string
	sub  []sub
}

func newHDX() *HDX {
	jar, err := cookiejar.New(nil)
	if err != nil {

	}
	client := http.Client{
		Transport: &http.Transport{

			Dial: func(netw, addr string) (net.Conn, error) {
				deadline := time.Now().Add(15 * time.Second)
				c, err := net.DialTimeout(netw, addr, time.Second*10)
				if err != nil {
					return nil, err
				}
				c.SetDeadline(deadline)
				return c, nil
			},
			// Proxy: func(_ *http.Request) (*url.URL, error) {
			// 	return url.Parse("http://127.0.0.1:8888")
			// },
			//DisableKeepAlives: true,
		},
		Jar: jar,
	}
	return &HDX{

		userName:   "97791319@qq.com",
		passWord:   "97791319aaqq",
		Client:     client,
		Cookies:    []*http.Cookie{},
		courseData: map[string][]class{},
		//courses:  []Course{},
	}

}
func (hdx *HDX) encryptPassWord(passWord string) (sign string, err error) {
	key := make([]byte, 24)
	buf := new(bytes.Buffer)
	decoder_buf, _ := base64.StdEncoding.DecodeString("DrZPGgL9WHkZrVQ0DT2bASoZE0Z8oc4s")

	err = binary.Write(buf, binary.BigEndian, decoder_buf)

	if err != nil {
		return "", err
	}
	buf.Read(key)
	en, err := TripleEcbDesEncrypt([]byte(passWord), key)
	if err != nil {
		return "", err
	}
	base64 := base64.StdEncoding.EncodeToString(en)
	return base64, nil
}
func (hdx *HDX) login(userName string, passWord string) (err error) {
	encryptedPassWord, err := hdx.encryptPassWord(passWord)
	if err != nil {
		return err

	}
	resp, err := hdx.Client.PostForm("http://api.cnmooc.org/v1",
		url.Values{
			"cmd":      {"sys.login"},
			"client":   {"cnmooc"},
			"user":     {userName},
			"password": {encryptedPassWord + "\n"},
			"sign":     {hdx.createSign("sys.logincnmooc" + userName + encryptedPassWord + "\n")},
		})

	if err != nil {
		return errors.New("Request Login Error!!!")
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.New("Read Body Error!!!")
	}
	defer resp.Body.Close()
	if retCode, err := jsonparser.GetString(body, "message"); (retCode == "命令执行成功") && (err == nil) {
		fmt.Println("登陆成功")

	} else {
		return errors.New(retCode)
	}

	return nil
}

func (hdx *HDX) createSign(data string) (out string) {
	h := md5.New()
	h.Write([]byte(data + "cnmooc@wisedu.com")) // 需要加密的字符串为 123456

	cipherStr := h.Sum(nil)
	//fmt.Println(hex.EncodeToString(cipherStr))
	return hex.EncodeToString(cipherStr)

}
func (hdx *HDX) parseCourse() (err error) {
	resp, err := hdx.Client.PostForm("http://api.cnmooc.org/v1",
		url.Values{
			"cmd":    {"course.my"},
			"client": {"cnmooc"},
			"index":  {"1"},
			"size":   {"5"},
			"status": {"10"},
			"sign":   {hdx.createSign("course.mycnmooc1510")},
		})

	defer resp.Body.Close()
	if err != nil {
		return errors.New("Request Login Error!!!")
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.New("Read Body Error!!!")
	}
	if retCode, err := jsonparser.GetString(body, "message"); (retCode == "命令执行成功") && (err == nil) {
		courseNum := 0
		fmt.Println("你的的课程:")
		jsonparser.ArrayEach(body, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			courseName, err := jsonparser.GetString(value, "courseName")
			courseID, err := jsonparser.GetString(value, "courseId")

			sessionID, err := jsonparser.GetString(value, "sessionId")
			//fmt.Println(courseID, err)

			hdx.courses = append(hdx.courses, Course{
				name:     courseName,
				courseId: courseID,
				sesson:   sessionID,
			})
			courseNum = courseNum + 1
		}, "result", "courses")

		for i, course := range hdx.courses {
			fmt.Println("\t" + strconv.Itoa(i) + ":" + course.name)
		}
	} else {
		return errors.New(retCode)
	}

	return nil
}

func (hdx *HDX) parseClass(courseId, session string) (err error) {
	resp, err := hdx.Client.PostForm("http://api.cnmooc.org/v1",
		url.Values{
			"all":     {"1"},
			"client":  {"cnmooc"},
			"cmd":     {"course.learn"},
			"course":  {courseId},
			"session": {session},
			"sign":    {hdx.createSign("course.learncnmooc" + courseId + session + "1")},
		})

	defer resp.Body.Close()
	if err != nil {
		return errors.New("Request Login Error!!!")
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.New("Read Body Error!!!")
	}

	if retCode, err := jsonparser.GetString(body, "message"); (retCode == "命令执行成功") && (err == nil) {
		courseNum := 0

		jsonparser.ArrayEach(body, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			unitName, err := jsonparser.GetString(value, "unitName")
			//fmt.Print(unitName)

			jsonparser.ArrayEach(value, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
				lessonName, err := jsonparser.GetString(value, "lessonName")
				//fmt.Print("\t" + lessonName + "\t")

				//fmt.Println(courseID, err)

				var item []string
				jsonparser.ArrayEach(value, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
					itemName, err := jsonparser.GetString(value, "itemName")
					itemUrl, err := jsonparser.GetString(value, "url")
					//fmt.Print("\t" + itemName + "\t")
					//fmt.Print("\t" + itemUrl + "\t")

					item = append([]string{itemName, itemUrl})

					var subData []sub
					jsonparser.ArrayEach(value, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
						label, err := jsonparser.GetString(value, "label")
						captionsUrl, err := jsonparser.GetString(value, "url")

						subData = append(subData, sub{
							name: label,
							url:  captionsUrl,
						})
						// fmt.Println(b)
						// fmt.Print("\t" + label + "\t")
						// fmt.Print("\t" + captionsUrl + "\t")

						//fmt.Println(courseID, err)

						//courseNum = courseNum + 1
					}, "captions")

					hdx.courseData[courseId] = append(hdx.courseData[courseId], class{
						path: []string{unitName, lessonName},
						name: itemName,
						url:  itemUrl,
						sub:  subData,
					})
					//fmt.Println(courseID, err)

					//courseNum = courseNum + 1
				}, "items")
				//courseNum = courseNum + 1
			}, "lessons")
			courseNum = courseNum + 1
		}, "result", "units")

	} else {
		return errors.New(retCode)
	}
	return nil
}

func (hdx *HDX) getTargetCourse() (err error) {

	var str string
	fmt.Print("你要爬取的课程:")
	fmt.Scanln(&str)

	num, err := strconv.Atoi(str)

	if err != nil {
		//fmt.Println("输入错误!!!")
		return errors.New("输入错误!!!")
	}

	if (0 > num) || (num >= len(hdx.courses)) {
		//fmt.Println("输入范围错误!!!")
		return errors.New("输入范围错误!!!")

	}

	hdx.target = append(hdx.target, num)
	fmt.Println("正在为你爬取", hdx.courses[num].name)

	return nil
}

func (hdx *HDX) download(path []string, name string, url string) (err error) {
	dir := filepath.Join(filepath.Dir(os.Args[0]), "Data")

	for _, i := range path {
		dir = filepath.Join(dir, i)

	}
	os.MkdirAll(dir, 0777)
	err = gorc.Download(url, dir+"\\"+name, name)
	// res, err := http.Get(url)
	// if err != nil {
	// 	return errors.New("下载错误")
	// }
	// defer res.Body.Close()
	// f, err := os.Create(dir + "\\" + name)
	// if err != nil {
	// 	return errors.New("创建文件错误")
	// }
	// io.Copy(f, res.Body)
	return nil

}

func (hdx *HDX) downloadTargetCourse() (err error) {
	for _, i := range hdx.target {
		fmt.Println("现在正在爬取", hdx.courses[i].name, "的课程列表")

		err = hdx.parseClass(hdx.courses[i].courseId, hdx.courses[i].sesson)
		if err != nil {
			return errors.New("爬取" + hdx.courses[i].name + "的课程列表出现了错误")
			//fmt.Println("爬取", hdx.courses[i], "的课程列表出现了错误")
		}

	}

	for _, i := range hdx.target {
		fmt.Println("现在正在下载课程", hdx.courses[i].name, "的课程列表")

		if courseData, ok := hdx.courseData[hdx.courses[i].courseId]; ok {

			for _, one := range courseData {
				fmt.Println("正在下载---", one.name)

				err = hdx.download(one.path, one.name+".mp4", one.url)
				if err != nil {
					return errors.New("下载" + one.name + "错误")
				}
				for _, sub := range one.sub {
					err = hdx.download(one.path, sub.name+"_"+one.name+".str", sub.url)
					if err != nil {
						return errors.New("下载" + one.name + "错误")
					}
				}

			}
		}

	}
	return nil
}

//ECB PKCS5Padding
func PKCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

//ECB PKCS5Unpadding
func PKCS5Unpadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

//Des加密
func encrypt(origData, key []byte) ([]byte, error) {
	if len(origData) < 1 || len(key) < 1 {
		return nil, errors.New("wrong data or key")
	}
	block, err := des.NewCipher(key)
	if err != nil {
		return nil, err
	}
	bs := block.BlockSize()
	if len(origData)%bs != 0 {
		return nil, errors.New("wrong padding")
	}
	out := make([]byte, len(origData))
	dst := out
	for len(origData) > 0 {
		block.Encrypt(dst, origData[:bs])
		origData = origData[bs:]
		dst = dst[bs:]
	}
	return out, nil
}
func decrypt(crypted, key []byte) ([]byte, error) {
	if len(crypted) < 1 || len(key) < 1 {
		return nil, errors.New("wrong data or key")
	}
	block, err := des.NewCipher(key)
	if err != nil {
		return nil, err
	}
	out := make([]byte, len(crypted))
	dst := out
	bs := block.BlockSize()
	if len(crypted)%bs != 0 {
		return nil, errors.New("wrong crypted size")
	}

	for len(crypted) > 0 {
		block.Decrypt(dst, crypted[:bs])
		crypted = crypted[bs:]
		dst = dst[bs:]
	}

	return out, nil
}

//[golang ECB 3DES Encrypt]
func TripleEcbDesEncrypt(origData, key []byte) ([]byte, error) {
	tkey := make([]byte, 24, 24)
	copy(tkey, key)
	k1 := tkey[:8]
	k2 := tkey[8:16]
	k3 := tkey[16:]

	block, err := des.NewCipher(k1)
	if err != nil {
		return nil, err
	}
	bs := block.BlockSize()
	origData = PKCS5Padding(origData, bs)

	buf1, err := encrypt(origData, k1)
	if err != nil {
		return nil, err
	}
	buf2, err := decrypt(buf1, k2)
	if err != nil {
		return nil, err
	}
	out, err := encrypt(buf2, k3)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func main() {

	hdx := newHDX()
	userName := ""
	passWord := ""
	fmt.Print("用户名:")

	fmt.Scanln(&userName)
	fmt.Print("密码:")
	fmt.Scanln(&passWord)
	fmt.Println()
	err := hdx.login(userName, passWord)

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	err = hdx.parseCourse()
	if err != nil {
		fmt.Println(err.Error())
	}
	err = hdx.getTargetCourse()
	if err != nil {
		fmt.Println(err.Error())
	}
	err = hdx.downloadTargetCourse()
	if err != nil {
		fmt.Println(err.Error())
	}
	os.Exit(1)

}
