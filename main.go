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
		"golang.org/x/image/bmp"

	//"image/png"
)

var CELL_SIZE int = 1201
var CELL_SWBD_SIZE int = 3601
var height_level []level
var water_level int16


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
}
type cell struct {
	data   *image.RGBA
	height []int16
	lat    int16
	lon    int16
}
type largeMap struct {
	domain mapRectangle
	data   *image.RGBA
}

func newLargeMap(domain mapRectangle) largeMap {
	var lm largeMap
	lm.domain = domain
	var width int = int(math.Floor(lm.domain.East-lm.domain.West)) * int(CELL_SIZE)
	var height int = int(math.Floor(lm.domain.North-lm.domain.South)) * int(CELL_SIZE)
	lm.data = image.NewRGBA(image.Rect(0, 0, width, height))
	return lm
}
func (lm *largeMap) setCell(c cell) {
	var x_offset int = int(math.Floor(float64(c.lon)-lm.domain.West)) * CELL_SIZE
	var y_offset int = int(math.Floor(lm.domain.North-float64(c.lat)-1)) * CELL_SIZE
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
			if x_offset+x < 0 {
				x = -x_offset
			} else if x_offset+x > width {
				break
			}
			(lm.data).SetRGBA(int(x_offset+x), int(y_offset+y), c.data.RGBAAt(x, y))
		}
	}
}
func saveImage(data image.Image, filename string) {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0666)

	if err != nil {
		log.Fatalln(err)
	}

	if err = bmp.Encode(f, data); err != nil {
		log.Fatalln(err)
	}
}
func (lm *largeMap) SaveImageLarge(filename string) {
	saveImage(lm.data, filename)
}
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}
func initCell(lat int16, lon int16) cell {
	var c cell
	c.data = image.NewRGBA(image.Rect(0, 0, CELL_SIZE, CELL_SIZE))
	c.lat = lat
	c.lon = lon
	for y := 0; y < CELL_SIZE; y++ {
		for x := 0; x < CELL_SIZE; x++ {
			cl := color.RGBA{0, 0, 0, 0}
			(c.data).SetRGBA(x, y, cl)
		}
	}
	return c
}
func heightToColor(height int16) color.RGBA {

	var num uint8 = 16

	for _, level := range height_level{
		if height >= level.Min && height <= level.Max {
			num = level.Bright
			break
		}
	}

	if height <= water_level {
		return color.RGBA{0, 0, num, 0}
	}
	return color.RGBA{0, num, 0, 0}
}
func hgtToCell(lat int16, lon int16, hgt *os.File, swbd []byte) cell {
	var c cell = initCell(lat, lon)
	var height int16
	 println(len(swbd))
	for y := 0; y < CELL_SIZE; y++ {
		for x := 0; x < CELL_SIZE; x++ {
			binary.Read(hgt, binary.BigEndian, &height)
			if swbd[(y*3)*CELL_SWBD_SIZE + (x*3)] == 0xff {
				height = -999
			}
			cl := heightToColor(height)
			(c.data).SetRGBA(x, y, cl)
		}
	}
	return c
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
func Download(lat int16, lon int16) cell {
	var filename_hgt string = fmt.Sprintf("N%02dE%03d.SRTMGL3.hgt", lat, lon)
	var filename_swbd string = fmt.Sprintf("N%02dE%03d.SRTMSWBD.raw", lat, lon)
	var url string = fmt.Sprintf("http://e4ftl01.cr.usgs.gov/SRTM/SRTMGL3.003/2000.02.11/%s.zip", filename_hgt)
	fmt.Printf("HGT file:%s \n", filename_hgt)
	if !FileExists(filename_hgt) {
		res, err := http.Get(url)
		if err != nil {
			fmt.Println(err)
		}
		defer res.Body.Close()
		if res.StatusCode == 404 {
			return initCell(lat, lon)
		}
		UnzipFirstfile(res.Body, res.ContentLength, filename_hgt,false)
	}


	var swbdData []byte
	url = fmt.Sprintf("http://e4ftl01.cr.usgs.gov/SRTM/SRTMSWBD.003/2000.02.11/%s.zip", filename_swbd)

	if !FileExists(fmt.Sprintf("%s.zip",filename_swbd)) {
		res, err := http.Get(url)
		if err != nil {
			log.Fatalln(err)
		}
		defer res.Body.Close()
		if res.StatusCode == 404 {
			return initCell(lat, lon)
		}
		swbdData, err = UnzipFirstfile(res.Body, res.ContentLength, "", true)
		if err != nil{
			log.Fatalln(err)
		}
	}else{
		var swbdFile *os.File
		swbdFile, err := os.Open(filename_swbd)
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

	return hgtToCell(lat, lon, hgtFile, swbdData)

	//return nil
}
func main() {

	dec := json.NewDecoder(os.Stdin)
	var jsonIn jsonData
	dec.Decode(&jsonIn)
	fmt.Printf("%+v\n", jsonIn)
	var area mapRectangle = jsonIn.Area

	height_level = jsonIn.Elevation.Level // global
	water_level = jsonIn.Elevation.Water  //global

	if area.North < area.South {
		fmt.Errorf("North lat is more south than South lat.")
	}
	if area.East < area.West {
		area.East += 360
	}
	var lm largeMap = newLargeMap(area)
	for lat := int16(math.Floor(area.South)); float64(lat) < area.North; lat++ {
		for lon := int16(math.Floor(area.West)); float64(lon) < area.East; lon++ {
			cell := Download(lat, lon)
			lm.setCell(cell)
		}
	}
	lm.SaveImageLarge("out2.bmp")
}
