package avlib

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

type WindAloftProductCode int

const (
	SixHourForcast        WindAloftProductCode = 1
	TwelveHourForcast     WindAloftProductCode = 3
	TwentyFourHourForcast WindAloftProductCode = 5
)

type WindsAloftProduct struct {
	Context WindsAloftProductContext `json:"@context"`
	Graph   []WindsAloftProductEntry `json:"@graph"`
}

type WindsAloftProductContext struct {
	Version string `json:"@version"`
	Vocab   string `json:"@vocab"`
}
type WindsAloftProductEntry struct {
	Id            string `json:"id"`
	CollectiveId  string `json:"wmoCollectiveId"`
	IssuingOffice string `json:"issuingOffice"`
	IssuanceTime  string `json:"issuanceTime"`
	ProductCode   string `json:"productCode"`
	ProductName   string `json:"productName"`
}

type WindsAloftDataProduct struct {
	Context       WindsAloftProductContext `json:"@context"`
	Id            string                   `json:"id"`
	CollectiveId  string                   `json:"wmoCollectiveId"`
	IssuingOffice string                   `json:"issuingOffice"`
	IssuanceTime  string                   `json:"issuanceTime"`
	ProductCode   string                   `json:"productCode"`
	ProductName   string                   `json:"productName"`
	ProductText   string                   `json:"productText"`
}

// Default get the 6 hour forecast
func GetWindsAloftProductText() (WindsAloftDataProduct, error) {
	return GetWindsAloftProductTextByType(SixHourForcast)
}
func GetWindsAloftProductTextByType(productCode WindAloftProductCode) (WindsAloftDataProduct, error) {
	var product WindsAloftProduct
	var productData WindsAloftDataProduct
	url := fmt.Sprintf("https://api.weather.gov/products/types/FD%d/locations/US%d", productCode, productCode)
	resp, err := http.Get(url)
	if err != nil {
		return WindsAloftDataProduct{}, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return WindsAloftDataProduct{}, err
	}
	err = json.Unmarshal(body, &product)
	if err != nil {
		return WindsAloftDataProduct{}, err
	}
	firstProductId := product.Graph[0].Id
	windsAloftUrl := fmt.Sprintf("https://api.weather.gov/products/%s", firstProductId)
	resp, err = http.Get(windsAloftUrl)
	if err != nil {
		return WindsAloftDataProduct{}, err
	}

	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return WindsAloftDataProduct{}, err
	}
	err = json.Unmarshal(body, &productData)
	if err != nil {
		return WindsAloftDataProduct{}, err
	}

	return productData, nil
}

func ExtractComponents(productText string) (string, string, []string, error) {
	exp, _ := regexp.Compile("^FT.*")
	validLineExp, _ := regexp.Compile("^VALID.*TEMP.*NEG.*")
	productTextLines := strings.Split(productText, "\n")
	validFromIndex := -1
	headerIndex := -1
	for i, s := range productTextLines {
		if exp.MatchString(s) {
			headerIndex = i
		}
		if validLineExp.MatchString(s) {
			validFromIndex = i
		}
	}
	if headerIndex == -1 || validFromIndex == -1 {
		return "", "", []string{}, fmt.Errorf("product text formatting issue, header index: %d valid from index: %d the following text was not formatted as expedted\n%s", headerIndex, validFromIndex, productText)
	}
	//fmt.Printf("Header Index: %d Valid From Index: %d\n", headerIndex, validFromIndex)
	return productTextLines[validFromIndex], productTextLines[headerIndex], productTextLines[headerIndex+1:], nil
}

func GetWindsAloftDataByCode(productCode WindAloftProductCode) (WindAloftData, error) {
	productData, err := GetWindsAloftProductTextByType(productCode)
	if err != nil {
		return WindAloftData{}, err
	}
	validHeader, altitudeHeader, windTempData, err := ExtractComponents(string(productData.ProductText))
	if err != nil {
		return WindAloftData{}, err
	}

	validLineInfo := ExtractValidLineInfo(validHeader)
	altitudeHeaderInfo := ProcessAltitudeHeader(altitudeHeader)
	windAloftData := WindAloftData{
		ValidInfo:            validLineInfo,
		AltitudeHeaderInfo:   altitudeHeaderInfo,
		IssuanceTime:         productData.IssuanceTime,
		LocationWindTempData: map[string]LocationWindTempData{},
	}
	for _, row := range windTempData {
		if len(row) == 0 {
			continue
		}
		location := row[:3]
		//fmt.Printf("Location Row: %s\n", row)
		locWindTempData := LocationWindTempData{
			AltitudeWindTemp: map[int]WindTempData{},
		}
		for _, altInf := range altitudeHeaderInfo {
			entryText := row[altInf.StartIdx:altInf.EndIdx]
			windTempData := ProcessWindTempEntry(entryText, altInf.Altitude, validLineInfo.NegAbove)
			locWindTempData.AltitudeWindTemp[altInf.Altitude] = windTempData
		}
		windAloftData.LocationWindTempData[location] = locWindTempData
	}

	return windAloftData, nil
}

func ProcessWindTempEntry(entry string, altitude int, negAbove int) WindTempData {
	if len(entry) == 0 {
		return WindTempData{
			WindDirectionDeg: math.MaxFloat64,
			WindSpeedKts:     math.MaxFloat64,
			TempC:            0,
			Altitude:         altitude,
		}
	}
	//fmt.Printf("Entry: %s\n", entry)
	direction, _ := strconv.ParseInt(entry[:2], 10, 0)
	speed, _ := strconv.ParseInt(entry[2:4], 10, 0)
	if direction > 40 && direction != 99 {
		direction = direction - 50
		speed += 100
	}
	direction = direction * 10
	if direction == 990 {
		direction = -1
	}
	temp := math.MaxFloat64
	if len(entry) > 4 {
		modifier := -1
		if altitude <= negAbove && strings.Contains(entry, "+") {
			modifier = 1
		}
		temp, _ = strconv.ParseFloat(entry[len(entry)-2:], 64)
		temp = temp * float64(modifier)

	}
	return WindTempData{
		WindDirectionDeg: float64(direction),
		WindSpeedKts:     float64(speed),
		TempC:            temp,
		Altitude:         altitude,
	}
}

func ExtractValidLineInfo(validLine string) ValidLineInfo {
	vexp, _ := regexp.Compile(`^VALID\s+(\d{6})Z\s+FOR USE\s+(\d{4})-(\d{4})Z.*(\d{4,6})$`)
	matches := vexp.FindStringSubmatch(validLine)
	negAbove, _ := strconv.ParseInt(matches[4], 10, 0)
	return ValidLineInfo{
		Valid:      matches[1],
		ForUseFrom: matches[2],
		ForUseTo:   matches[3],
		NegAbove:   int(negAbove),
	}
}

func ProcessAltitudeHeader(header string) []DataHeaderInfo {
	var dhis []DataHeaderInfo
	fields := strings.Split(header, " ")
	lastIndex := 3
	for _, f := range fields {
		if len(f) > 1 && f != "FT" {
			fidx := strings.Index(header, f)
			alt, _ := strconv.ParseInt(strings.TrimSpace(f), 10, 0)
			dhis = append(dhis, DataHeaderInfo{
				Altitude: int(alt),
				StartIdx: lastIndex + 1,
				EndIdx:   fidx + len(f),
			})
			lastIndex = fidx + len(f)
		}
	}
	return dhis
}

type WindAloftData struct {
	ValidInfo            ValidLineInfo
	AltitudeHeaderInfo   []DataHeaderInfo
	IssuanceTime         string
	LocationWindTempData map[string]LocationWindTempData
}

type LocationWindTempData struct {
	AltitudeWindTemp map[int]WindTempData
}

type WindTempData struct {
	WindDirectionDeg float64
	WindSpeedKts     float64
	TempC            float64
	Altitude         int
}

type ValidLineInfo struct {
	Valid      string
	ForUseFrom string
	ForUseTo   string
	NegAbove   int
}

type DataHeaderInfo struct {
	Altitude int
	StartIdx int
	EndIdx   int
}
