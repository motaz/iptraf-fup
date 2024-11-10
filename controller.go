package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/motaz/codeutils"
	"github.com/motaz/redisaccess"
)

var Console_Reset = "\033[0m"
var Console_Red = "\033[31m"
var Console_Green = "\033[32m"
var Console_Yellow = "\033[33m"
var Console_Blue = "\033[34m"
var Console_Console_Magenta = "\033[35m"
var Console_Cyan = "\033[36m"
var Console_Gray = "\033[37m"
var Console_White = "\033[97m"

func getConfigValue(paramname, defaultvalue string) (value string) {
	value = codeutils.GetConfigWithDefault("config.ini", paramname, defaultvalue)
	return
}

func writeLog(event string, display bool) {

	if display {
		fmt.Println(event)
	}
	codeutils.WriteToLog(event, "iptraf")
}

type DeviceType struct {
	Mac       string
	Total     int64
	Generated string
	Day       string
}

func searchDevice(mac, day string, list []DeviceType) (found bool, index int) {

	found = false
	for i, item := range list {
		if mac == item.Mac && day == item.Day {
			found = true
			index = i
			break
		}
	}
	return
}

func addDevice(device DeviceType, list *[]DeviceType) {

	found, index := searchDevice(device.Mac, device.Day, *list)
	if found {
		(*list)[index] = device
	} else {
		*list = append(*list, device)
	}
}

func addToGrandTotal(GrandList *[]DeviceType, subList []DeviceType) {

	for _, item := range subList {

		found, index := searchDevice(item.Mac, item.Day, *GrandList)
		if found {

			(*GrandList)[index].Total += item.Total
			(*GrandList)[index].Generated = item.Generated

		} else {
			*GrandList = append(*GrandList, item)

		}
	}
}

func checkSkip(skipList []string, mac string) (skip bool) {

	skip = false
	mac = strings.ToLower(mac)
	if len(skipList) > 0 {
		for _, item := range skipList {
			if strings.ToLower(item) == mac {
				skip = true
				break
			}
		}
	}
	return
}

func parseLimit(limitStr string) (limit int64) {

	limitStr = strings.ReplaceAll(limitStr, " ", "")
	mtype := strings.ToUpper(limitStr)
	mtype = mtype[len(mtype)-1:]
	if strings.Contains("KMG", mtype) {
		limitStr = limitStr[:len(limitStr)-1]
		limitInt, _ := strconv.Atoi(limitStr)
		switch mtype {
		case "K":
			limit = int64(limitInt) * 1024
		case "M":
			limit = int64(limitInt) * 1024 * 1024
		case "G":
			limit = int64(limitInt) * 1024 * 1024 * 1024
		}

	} else {
		limit, _ = strconv.ParseInt(limitStr, 10, 64)
	}
	return
}

func getMacLimit(mac string) (limit int64) {

	limitStr := codeutils.GetConfigValue("mac.ini", mac)
	if strings.TrimSpace(limitStr) == "" {
		limitStr = getConfigValue("limit", "500m")
	}
	limit = parseLimit(limitStr)
	return
}

func updateTraffic(list []DeviceType) {

	skipStr := strings.ReplaceAll(getConfigValue("skiplist", ""), " ", "")
	skipList := strings.Split(skipStr, ",")
	var color string
	var measure string
	for _, item := range list {
		used := codeutils.FormatFloatCommas(float64(item.Total)/1024/1024/1024, 1)
		measure = "G"
		color = Console_Cyan
		if strings.HasPrefix(used, "0.") {
			used = codeutils.FormatFloatCommas(float64(item.Total/1024/1024), 0)
			measure = "M"
			color = Console_Yellow
		}
		if used == "0" {
			used = codeutils.FormatFloatCommas(float64(item.Total/1024), 0)
			measure = "K"
			color = Console_Green
		}
		fmt.Printf("%s %10s%s%s%s\n", item.Mac, used, color, measure, Console_Reset)
		writeLog(fmt.Sprintf("%s %10s%s", item.Mac, used, measure), false)
		skip := checkSkip(skipList, item.Mac)
		if skip {
			writeLog("----------------^-------     Skipped..", true)
		} else {
			limit := getMacLimit(item.Mac)
			if item.Total > limit {

				err := blockMac(item)
				if err != nil {
					writeLog(err.Error(), true)
				}
			}
		}

	}
}

func process() {

	filename := getConfigValue("logfile", "")
	file, err := os.OpenFile(filename, os.O_RDONLY, 0644)

	if err == nil {

		defer file.Close()

		scanner := bufio.NewScanner(file)
		counter := 0
		generated := ""
		var grandList []DeviceType
		grandList = make([]DeviceType, 0)

		var list []DeviceType
		list = make([]DeviceType, 0)

		for scanner.Scan() {

			line := scanner.Text()
			if strings.Contains(line, "log, generated") {
				generated = line[strings.Index(line, "generated")+10:]
				generated = strings.TrimSpace(generated)
			}
			if strings.Contains(line, "address:") {
				mac := line[strings.Index(line, "address:")+8:]
				mac = strings.TrimSpace(mac)
				//	fmt.Println(mac)
				var total int64 = 0
				for range 2 {
					scanner.Scan()
					line = scanner.Text()
					//fmt.Println(line)
					if strings.Contains(line, ";") {
						bytesStr := line[strings.Index(line, ",")+1 : strings.Index(line, ";")-1]
						bytesStr = bytesStr[:strings.Index(bytesStr, "byte")-1]
						bytesStr = strings.TrimSpace(bytesStr)
						//	fmt.Println("Bytes:", bytesStr)
						var bytes int64
						bytes, err = strconv.ParseInt(bytesStr, 10, 64)
						if err != nil {
							fmt.Println(err.Error())
						}
						total += bytes
					}
				}
				//fmt.Println("Total: ", total)
				var device DeviceType
				if total > 2048 && len(generated) > 10 {
					device.Mac = mac
					device.Total = total
					device.Generated = generated
					device.Day = generated[:strings.Index(generated, " ")]

					//	fmt.Printf("%+v\n", device)
					addDevice(device, &list)
				}
			}
			if counter > 0 && strings.Contains(line, "monitor started") {

				addToGrandTotal(&grandList, list)
			} else {
				//	fmt.Println(line)
			}
			counter++

		}
		addToGrandTotal(&grandList, list)
		fmt.Println("--------------------------------------")
		updateTraffic(grandList)
	} else {

		writeLog("Error in opening traffic file: "+err.Error(), true)

	}

}

func blockMac(device DeviceType) (err error) {

	client, err := redisaccess.InitRedis("localhost", "")
	if err != nil {
		writeLog("Error in Redis: "+err.Error(), true)
	}
	key := "iptraf::" + device.Day + "::" + device.Mac
	_, found, err := redisaccess.GetValue(key)
	if !found {
		writeLog(fmt.Sprintf("%s has exceeded limit %s ",
			device.Mac, codeutils.FormatFloatCommas(float64(device.Total), 0)), true)
		_, errStr := shell("/sbin/iptables -A INPUT -j DROP -m mac --mac-source " + device.Mac)
		writeLog("Blocking: "+device.Mac+" "+errStr, true)
		if errStr == "" {
			defer client.Close()
			redisaccess.SetValue(key, device, time.Hour*12)

		} else {
			err = errors.New(errStr)
		}
	} else {
		writeLog("----------------^-------     Already blocked", true)
	}
	return
}

func shell(command string) (result string, err string) {

	var out bytes.Buffer
	var errBuf bytes.Buffer

	cmd := exec.Command("/bin/bash", "-c", command)
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	cmd.Run()
	result = out.String()
	err = errBuf.String()
	return
}
