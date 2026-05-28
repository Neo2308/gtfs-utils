package apiDataFetcher

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Neo2308/gtfs-utils/fileUtils"
)

var throttleTriggerTime = map[string]time.Time{}

type ApiDataFetcher[T any] struct {
	DataLocation     *T
	UrlType          string
	CacheFileName    string
	Retries          int
	AlwaysRefetch    bool
	RefetchFunc      func() error
	ThrottleInterval time.Duration
}

func NewApiDataFetcher[T any](dataLocation *T, urlType string, cacheFileName string, refetchFunc func() error, alwaysRefetch bool, retries int, throttleIntervalMs int) *ApiDataFetcher[T] {
	return &ApiDataFetcher[T]{
		DataLocation:     dataLocation,
		UrlType:          urlType,
		CacheFileName:    cacheFileName,
		AlwaysRefetch:    alwaysRefetch,
		RefetchFunc:      refetchFunc,
		Retries:          retries,
		ThrottleInterval: time.Duration(throttleIntervalMs) * time.Millisecond,
	}
}

func (a *ApiDataFetcher[T]) LoadData() error {
	// file_name := fmt.Sprintf("%s.json", fmt.Sprintf("%d", t.trainNumber))
	// file_name := '{}.json'.format(train_number)
	// Load json from cache if available
	data, _ := fileUtils.LoadFile(a.CacheFileName, fileUtils.CACHE)
	return json.Unmarshal(data, a.DataLocation)
}

func (a *ApiDataFetcher[T]) PopulateData() error {
	if a.AlwaysRefetch {
		fmt.Printf("Cache disabled for %s, fetching from API...\n", a.CacheFileName)
		return a.RefetchFunc()
	}
	// Load json from cache if available
	data, err := fileUtils.LoadFile(a.CacheFileName, fileUtils.CACHE)
	if err == nil {
		// fmt.Printf("Cache hit for %s, loading from cache...\n", a.CacheFileName)
		// fmt.Println(string(data))
		var temp T
		return json.Unmarshal(data, &temp)
	}
	if !os.IsNotExist(err) {
		return err
	}
	fmt.Printf("Cache miss for %s, fetching from API...\n", a.CacheFileName)
	return a.RefetchFunc()
}

func (a *ApiDataFetcher[T]) FetchData(req *http.Request, client *http.Client, reqFunc func() *http.Request) error {
	a.throttleRequests()
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == 504 {
		fmt.Println("Request timed out, retrying...")
		res, err = client.Do(req)
		if err != nil {
			fmt.Println(err)
			return err
		}
	} else if res.StatusCode == 500 {
		for i := 0; res.StatusCode == 500 && i < a.Retries; i++ {
			fmt.Printf("Internal server error, retrying (failed %d times)...\n", i)
			client = &http.Client{}
			req = reqFunc()
			res, err = client.Do(req)
			if err != nil {
				fmt.Println(err)
				return err
			}
		}
	}
	if res.StatusCode == 200 {
		// fmt.Printf("%+v", res.Body)
		var result interface{}
		json.NewDecoder(res.Body).Decode(&result)
		defer res.Body.Close()
		// fmt.Printf("%+v", result)
		responseJson, _ := json.MarshalIndent(result, "", "    ")
		return fileUtils.SaveFile(a.CacheFileName, responseJson, fileUtils.CACHE)
	}
	return fmt.Errorf("failed to fetch %s data, %d - %s", a.UrlType, res.StatusCode, res.Status)
}

func (a *ApiDataFetcher[T]) throttleRequests() {
	if _, ok := throttleTriggerTime[a.UrlType]; !ok {
		throttleTriggerTime[a.UrlType] = time.Now()
		return
	}
	fmt.Printf("Checking if request for %s should be throttled at %v \n", a.UrlType, time.Now())
	// fmt.Printf("Checking this %v %v %v\n", a.UrlType, time.Now(), throttleTriggerTime[a.UrlType], throttleTriggerTime[a.UrlType].Add(throttlingInterval))
	if !time.Now().After(throttleTriggerTime[a.UrlType].Add(a.ThrottleInterval)) {
		sleepDuration := throttleTriggerTime[a.UrlType].Add(a.ThrottleInterval).Sub(time.Now())
		fmt.Printf("Throttling request for %s, sleeping for %v seconds...\n", a.UrlType, sleepDuration.Seconds())
		time.Sleep(sleepDuration)
	}
	throttleTriggerTime[a.UrlType] = time.Now()
}
