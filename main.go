package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

var domains = []string{
	"github.com",
	"gist.github.com",
	"assets-cdn.github.com",
	"raw.githubusercontent.com",
	"gist.githubusercontent.com",
	"cloud.githubusercontent.com",
	"camo.githubusercontent.com",
	"avatars.githubusercontent.com",
	"avatars0.githubusercontent.com",
	"avatars1.githubusercontent.com",
	"avatars2.githubusercontent.com",
	"avatars3.githubusercontent.com",
	"avatars4.githubusercontent.com",
	"avatars5.githubusercontent.com",
	"avatars6.githubusercontent.com",
	"avatars7.githubusercontent.com",
	"avatars8.githubusercontent.com",
	"github.githubassets.com",
}

var startTag = "# GitHub Start\r\n"
var endTag = "# GitHub End\r\n"

func main() {
	var mode int
	fmt.Println("1. Create the host file to current directory (default)")
	fmt.Println("2. Append the host to system host file in linux (/etc/hosts)")
	fmt.Println("3. Append the host to system host file in windows (/mnt/c/Windows/System32/drivers/etc/hosts)")
	for {
		fmt.Print("Choose the mode [1-3]: ")
		fmt.Scanln(&mode)
		if mode >= 0 && mode <= 3 {
			break
		}
	}
	filePath := ""
	if mode == 1 || mode == 0 {
		filePath = "hosts"
		WriteHostToFile("", filePath)
		return
	} else if mode == 2 {
		filePath = "/etc/hosts"
	} else {
		filePath = "/mnt/c/Windows/System32/drivers/etc/hosts"
	}
	filePathBak := filePath + "_bak"

	// backup
	Copy(filePathBak, filePath)

	// read content
	in, err := os.Open(filePath)
	if err != nil {
		fmt.Println("open file fail:", err)
		os.Exit(-1)
	}
	defer in.Close()
	reader := bufio.NewReader(in)
	var strSlice []string
	line := 0
	startLine := 0
	endLine := 0
	for {
		line = line + 1
		str, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		strSlice = append(strSlice, str)
		if str == startTag {
			startLine = line
		} else if str == endTag {
			endLine = line
		}
	}
	if startLine > 0 && endLine > 0 {
		strSlice = append(strSlice[:startLine-1], strSlice[endLine:]...)
	}
	str := strings.Join(strSlice, "")

	// write content
	WriteHostToFile(str, filePath)
}

type HostChan struct {
	Domain string
	Ip     string
	Err    error
}

func WriteHostToFile(str string, filePath string) {
	str += startTag
	ch := make(chan *HostChan)
	for _, v := range domains {
		go httpPostForm(v, ch)
	}
	fmt.Println("================\nstart get host：\n================")
	hostMap := make(map[string]string)
	for range domains {
		chRec := <-ch
		if chRec.Err != nil {
			fmt.Println(chRec.Err.Error() + " " + chRec.Domain)
			return
		}
		hostMap[chRec.Domain] = chRec.Ip
		fmt.Println(chRec.Ip + " " + chRec.Domain)
	}
	for _, v := range domains {
		str += hostMap[v] + " " + v + "\r\n"
	}

	str += endTag
	out, err := os.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println("open file fail:", err)
		return
	}
	defer out.Close()
	writer := bufio.NewWriter(out)
	writer.WriteString(str)
	writer.Flush()
	fmt.Println("================\ndone\n================")
}

func Copy(dstName, srcName string) (written int64, err error) {
	src, err := os.Open(srcName)
	if err != nil {
		return
	}
	defer src.Close()
	dst, err := os.OpenFile(dstName, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	defer dst.Close()
	return io.Copy(dst, src)
}

func httpPostForm(domain string, ch chan<- *HostChan) {
	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://www.ipaddress.com/ip-lookup", strings.NewReader("host="+domain))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("UserAgent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		ch <- &HostChan{Domain: domain, Err: err}
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		ch <- &HostChan{Domain: domain, Err: err}
		return
	}
	shortStr := string(body)[12000:30000]
	index := strings.Index(shortStr, " ("+domain+")")
	var res string
	if index > 0 {
		strStart := "IP Lookup : "
		indexStart := strings.Index(shortStr, strStart)
		res = shortStr[indexStart+len(strStart) : index]
	} else {
		strStart := "<input name=\"host\" type=\"radio\" value=\""
		indexStart := strings.Index(shortStr, strStart)
		indexEnd := strings.Index(shortStr[indexStart+len(strStart):indexStart+len(strStart)+100], "\"")
		if indexEnd > 0 {
			res = shortStr[indexStart+len(strStart) : indexStart+len(strStart)+indexEnd]
		} else {
			ch <- &HostChan{Domain: domain, Err: errors.New("get indexEnd error")}
		}
	}
	if res == "" {
		ch <- &HostChan{Domain: domain, Err: errors.New("empty host")}
		return
	}
	ch <- &HostChan{Domain: domain, Ip: res, Err: nil}
	return
}
