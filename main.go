package main

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/robfig/cron/v3"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/spf13/viper"
	"os/exec"
	"strings"
)

func main() {

	viper.SetConfigName("config") // 设置配置文件名 (不带后缀)
	viper.AddConfigPath(".")      // 第一个搜索路径
	err := viper.ReadInConfig()   // 读取配置数据
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
	fmt.Println(viper.AllSettings())
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("配置发生变更：", e.String())
	})

	c := cron.New()
	fmt.Println(c)
	spec := "*/10 * * * *"    // 每一分钟， 现在v3 的版本是 到分钟
	c.AddFunc(spec, checkVersion)
	c.Start()
	select{}
}
type Version struct {
	ID        uint      `json:"id" gorm:"primary_key"`
	Version   uint `json:"version"`
	Description string `json:"description"`
	URL string `json:"url"`
	MD5 string `json:"md_5"`
	SHA1 string `json:"sha1"`
}
func checkVersion() {
	c := http.Client{}
	resp,err := c.Get(viper.GetString("update_url"))
	if err !=nil {
		return
	}
	var version Version
	body,err := ioutil.ReadAll(resp.Body)
	if err !=nil {
		return
	}
	err = json.Unmarshal(body, &version)
	if err !=nil {
		return
	}
	if version.Version > viper.GetUint("version") { //更新
		if isProcessExist(){
			err := killProcess()
			if err != nil {
				return
			}
		}
		err = httpGetFile(version.URL, viper.GetString("path") +string(os.PathSeparator)+viper.GetString("app_name"))
		if err !=nil {
			return
		}
		err = startProcess()
		if err !=nil {
			log.Println(err)
			return
		}
		viper.Set("version", version.Version)
		viper.WriteConfig()
	}
}

func getsha1() (string,error) {
	f, e := os.Open(viper.GetString("path") +path.Ext("/")+ string(os.PathSeparator)+ viper.GetString("app_name"))
	if e != nil {
		return "",e
	}
	h := sha1.New()
	_, e = io.Copy(h, f)
	if e != nil {
		return "",e
	}
	return hex.EncodeToString(h.Sum(nil)),nil
}

func getmd5() string {
	md5 := md5.New()
	file, _ := os.Open(viper.GetString("path") +path.Ext("/")+ string(os.PathSeparator)+ viper.GetString("app_name"))
	io.Copy(md5,file)
	return  hex.EncodeToString(md5.Sum(nil))
}

func isProcessExist() bool {
	cmd := exec.Command("tasklist")
	output, _ := cmd.Output()
	fields := strings.Fields(string(output))
	for _, v := range fields {
		if v == viper.GetString("app_name")  {
			return true
		}
	}
	return false
}

func startProcess() error {
	path := viper.GetString("path")
	c := `cd ` + path + `
	` + `start .\\` + viper.GetString("app_name") //
	cmd := exec.Command("powershell.exe", "/c", c)
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func killProcess() error {
	cmd := exec.Command("taskkill", "-f", "-im",  viper.GetString("app_name"))
	_, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	return nil
}

func httpGetFile(url, path string) error {
	res, err := http.Get(url)

	if err != nil {
		return err
	}
	f, err := os.Create(path)
	defer f.Close()
	if err != nil {
		return err
	}
	_, err = io.Copy(f, res.Body)
	return err
}