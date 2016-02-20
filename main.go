package main

import (
	"fmt"
	"os"

	"archive/zip"
	"encoding/binary"
	"encoding/json"
	"image"
	"image/color"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
//		"golang.org/x/image/bmp"

	"image/png"
	"bytes"
	"strings"
)

var CELL_SIZE int = 1201
var CELL_SWBD_SIZE int = 3601
var elevation_level []level
var water_level int16
var drawing_style drawing
var margin_style margin
var area mapRectangle
var lm largeMap

type drawing int8
const(
	Degree drawing =iota
	Mercator
)
type margin int8
const(
	Fill margin = iota
	Water
)
type mapRectangle struct {
	North float64
	East  float64
	West  float64
	South float64
}
type level struct{
	Max int16
	Min int16
	Bright uint8
}
type elevation struct{
	Water int16
	Level []level
}
type jsonData struct{
	Area mapRectangle
	Elevation elevation
	Filename string
	Drawing string
	Margin string
}
type elevationData struct{
	data []int16
	width int
	lat int16
	lon int16
	received bool
}
type cell struct {
	data   *image.RGBA
	elevation []int16
	lat    int16
	lon    int16
}
type largeMap struct {
	domain mapRectangle
	data   *image.RGBA
}

func newLargeMap(domain mapRectangle, width ,height int) largeMap {
	var lm largeMap
	lm.domain = domain

	lm.data = image.NewRGBA(image.Rect(0, 0, width, height))
	return lm
}
func (lm *largeMap) setCell(c cell) {

}
func saveImage(data image.Image, filename string) {

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalln(err)
	}
	if err = png.Encode(f, data); err != nil {
	//	log.Fatalln(err)
		fmt.Println(err.Error())
	}
}
func (lm *largeMap) SaveImageLarge(filename string) {
	saveImage(lm.data, filename)
}
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func elevationToColor(elevation int16) color.RGBA {

	var num uint8 = 16

	for _, level := range elevation_level{
		if elevation >= level.Min && elevation <= level.Max {
			num = level.Bright
			break
		}
	}

	if elevation <= water_level {
		return color.RGBA{0, 0, num, 255}
	}
	return color.RGBA{0, num, 0, 255}
}


type sliceReaderAt []byte

func (r sliceReaderAt) ReadAt(b []byte, off int64) (int, error) {
	copy(b, r[int(off):int(off)+len(b)])
	return len(b), nil
}



func UnzipFirstfile(body io.Reader, size int64, dest string, ret_byte bool) ([]byte,error) {
	//http://barsoom.seesaa.net/article/280192578.html
	b := make(sliceReaderAt, size)

	if _, err := io.ReadFull(body, b); err != nil {
		log.Fatalln(err)
		return nil,err
	}
	var rd *zip.Reader
	rd, err := zip.NewReader(b, size)
	if  err != nil {
		log.Fatalln(err)
		return nil,err
	}
	rc, err := rd.File[0].Open()
 	defer func() {
		rc.Close()
	}()
	buf := make([]byte, rd.File[0].UncompressedSize)
 	if _, err = io.ReadFull(rc, buf); err != nil {

		return nil,err
	}
 	if dest !="" {
		if err = ioutil.WriteFile(dest, buf, 666); err!=nil{
			return nil,err
		}
	}

	if ret_byte == true{
		return buf,nil
	}else{
		return nil,nil
	}
}
func Download(lat int16, lon int16) elevationData {
	var filename_hgt string = fmt.Sprintf("N%02dE%03d.SRTMGL3.hgt", lat, lon)
	var filename_swbd string = fmt.Sprintf("N%02dE%03d.SRTMSWBD.raw", lat, lon)
	var url string = fmt.Sprintf("http://e4ftl01.cr.usgs.gov/SRTM/SRTMGL3.003/2000.02.11/%s.zip", filename_hgt)
	var cellElevationData elevationData
	cellElevationData.data = make([]int16,CELL_SIZE*CELL_SIZE)
	cellElevationData.width = CELL_SIZE
	cellElevationData.lat = lat
	cellElevationData.lon = lon
	cellElevationData.received = false

	fmt.Printf("\nLat:%d Lon:%d\n", lat,lon)
	if !FileExists(filename_hgt) {
		fmt.Printf(" Download hgt...\n")
		res, err := http.Get(url)
		if err != nil {
			fmt.Println(err)
		}
		defer res.Body.Close()
		if res.StatusCode == 404 {
			fmt.Println("hgt not found")
			cellElevationData.received = false
			return cellElevationData
		}
		UnzipFirstfile(res.Body, res.ContentLength, filename_hgt,false)
	}


	var swbdData []byte
	url = fmt.Sprintf("http://e4ftl01.cr.usgs.gov/SRTM/SRTMSWBD.003/2000.02.11/%s.zip", filename_swbd)
	filename_swbd_zipped := fmt.Sprintf("%s.zip",filename_swbd)
	if !FileExists(filename_swbd_zipped) {
		fmt.Println("Download SWBD...")
		res, err := http.Get(url)
		if err != nil {
			log.Fatalln(err)
		}
		defer res.Body.Close()
		if res.StatusCode == 404 {
			cellElevationData.received = false
			return cellElevationData
		}
		var swbdDataZipped []byte

		swbdDataZipped, err = ioutil.ReadAll(res.Body)
		if err  != nil{
			log.Fatalln(err)
		}
		err = ioutil.WriteFile(filename_swbd_zipped,swbdDataZipped,666)

		if err!=nil{
			log.Fatalln(err)
		}
		swbdData, err = UnzipFirstfile(bytes.NewReader(swbdDataZipped), res.ContentLength, "", true)

		if err != nil{
			log.Fatalln(err)
		}
	}else{
		var swbdFile *os.File
		swbdFile, err := os.Open(filename_swbd_zipped)
		if  err!=nil{
			log.Fatalln(err)
		}
		fi, err := swbdFile.Stat()
		if err != nil{
			log.Fatalln(err)
		}
		swbdData,err = UnzipFirstfile(swbdFile,fi.Size(),"",true)
		if err != nil{
			log.Fatalln(err)
		}
	}

	var hgtFile *os.File
	var err error
	if hgtFile, err = os.Open(filename_hgt); err != nil {
		log.Fatalln(err)
	}
	var elevation int16
	 cellElevationData.received = true
	for y := 0; y < CELL_SIZE; y++ {
		for x := 0; x < CELL_SIZE; x++ {
			binary.Read(hgtFile, binary.BigEndian, &elevation)
			if swbdData[(y*3)*CELL_SWBD_SIZE + (x*3)] == 0xff {
				elevation = water_level
			}
			cellElevationData.data[y*cellElevationData.width + x] = elevation
		}
	}
	return cellElevationData

}
func degreeMap(){
	var mapDomain mapRectangle = area
	var width int = int(math.Floor( ( mapDomain.East-mapDomain.West) * float64(CELL_SIZE) ))
	var height int = int(math.Floor( (mapDomain.North-mapDomain.South) * float64(CELL_SIZE) ))
	lm = newLargeMap(area,width,height)
	println(lm.data.Bounds().Dx())
	for lat := int16(math.Floor(mapDomain.South)); float64(lat) < mapDomain.North; lat++ {
		for lon := int16(math.Floor(mapDomain.West)); float64(lon) <mapDomain.East; lon++ {
			elevationData := Download(lat, lon)

			var x_offset int = int(math.Floor( ( float64(elevationData.lon)-mapDomain.West) * float64(CELL_SIZE) ))
			var y_offset int = int(math.Floor( (mapDomain.North-float64(elevationData.lat)-1) * float64(CELL_SIZE) ))
			width := int(lm.data.Bounds().Dx())
			height := int(lm.data.Bounds().Dy())
			var x, y int
			for y = 0; y < CELL_SIZE; y++ {
				if y_offset+y < 0 {
					y = -y_offset
				} else if y_offset+y > height {
					break
				}
				for x = 0; x < CELL_SIZE; x++ {
					if x_offset + x < 0 {
						x = -x_offset
					} else if x_offset + x > width {
						break
					}
					var elevation int16

					if elevationData.received {
						elevation = elevationData.data[y * elevationData.width + x]
					}else{
						elevation = math.MinInt16
					}
					cl := elevationToColor(elevation)
					(lm.data).SetRGBA(int(x_offset+x), int(y_offset+y),cl)
				}
			}
		}
	}
}
func mercatorMap(){

}
func main() {

	dec := json.NewDecoder(os.Stdin)
	var jsonIn jsonData
	dec.Decode(&jsonIn)
	fmt.Printf("%+v\n", jsonIn)
	area  = jsonIn.Area

	elevation_level = jsonIn.Elevation.Level // global
	water_level = jsonIn.Elevation.Water  //global


	margin_type_string := strings.ToLower(jsonIn.Margin)
	switch margin_type_string {
	case "fill":
		margin_style = Fill
	case "water":
		margin_style = Water
	default:
		margin_style = Water
	}

	if area.North < area.South {
		fmt.Println("North lat is more south than South lat.")
		os.Exit(1)
	}
	if area.East < area.West {
		area.East += 360
	}
	drawing_type_string := strings.ToLower(jsonIn.Drawing)
	switch drawing_type_string {
	case "mercator":
		drawing_style = Mercator
		mercatorMap()
	case "degree":
		drawing_style = Degree
		degreeMap()
	default:
		drawing_style = Degree
		degreeMap()
	}

	lm.SaveImageLarge(jsonIn.Filename)
}
