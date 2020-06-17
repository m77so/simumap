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
	"flag"
	"bytes"
	"image/png"
	"strings"
)

var CELL_SIZE int = 1201
var CELL_DIV int = 1200
var CELL_SWBD_SIZE int = 3601
var CELL_SWBD_DIV int = 3600
var EARTH_RADIUS float64 = 6378137
var EARTH_FLATTENING float64 = 1 / 298.257222101
var EARTH_ECCENTRICITY float64 = 0.08181919104

var elevation_level []level
var water_level int16
var drawing_style drawing
var margin_style margin
var area mapRectangle
var lm largeMap

type drawing int8

const (
	Degree drawing = iota
	Mercator
)

type margin int8

const (
	Fill margin = iota
	Water
)

type mapRectangle struct {
	North float64
	East  float64
	West  float64
	South float64
}
type level struct {
	Max    int16
	Min    int16
	Bright uint8
}
type elevation struct {
	Water int16
	Level []level
}
type drawingStruct struct {
	Style     string
	Pixelsize float64
	Baselat   float64
	Margin    string
}
type jsonData struct {
	Area      mapRectangle
	Elevation elevation
	Filename  string
	Drawing   drawingStruct
}
type elevationData struct {
	data     []int16
	width    int
	lat      int16
	lon      int16
	received bool
}
type cell struct {
	data      *image.RGBA
	elevation []int16
	lat       int16
	lon       int16
}
type largeMap struct {
	domain mapRectangle
	data   *image.RGBA
}

func newLargeMap(domain mapRectangle, width, height int) largeMap {
	var lm largeMap
	lm.domain = domain

	lm.data = image.NewRGBA(image.Rect(0, 0, width, height))
	return lm
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

	for _, level := range elevation_level {
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

func UnzipFirstfile(body io.Reader, size int64, dest string, ret_byte bool) ([]byte, error) {
	//http://barsoom.seesaa.net/article/280192578.html
	b := make(sliceReaderAt, size)

	if _, err := io.ReadFull(body, b); err != nil {
		log.Fatalln(err)
		return nil, err
	}
	var rd *zip.Reader
	rd, err := zip.NewReader(b, size)
	if err != nil {
		log.Fatalln(err)
		return nil, err
	}
	rc, err := rd.File[0].Open()
	defer func() {
		rc.Close()
	}()
	buf := make([]byte, rd.File[0].UncompressedSize)
	if _, err = io.ReadFull(rc, buf); err != nil {

		return nil, err
	}
	if dest != "" {
		if err = ioutil.WriteFile(dest, buf, 666); err != nil {
			return nil, err
		}
	}

	if ret_byte == true {
		return buf, nil
	} else {
		return nil, nil
	}
}

func unzip(filename string)  ([]byte) {
	var fl *os.File
	var data []byte
	fl, err := os.Open(filename)
	if err != nil {
		log.Fatalln(err)
	}
	fi, err := fl.Stat()
	if err != nil {
		log.Fatalln(err)
	}
	data, err = UnzipFirstfile(fl, fi.Size(), "", true)
	if err != nil {
		log.Fatalln(err)
	}
	return data
}

func Download(lat int16, lon int16, dryrun bool) (elevationData, []byte) {
	var filename_hgt string = fmt.Sprintf("N%02dE%03d.SRTMGL3.hgt", lat, lon)
	var filename_swbd string = fmt.Sprintf("N%02dE%03d.SRTMSWBD.raw", lat, lon)
	
	var url string = fmt.Sprintf("https://e4ftl01.cr.usgs.gov/MEASURES/SRTMGL3.003/2000.02.11/%s.zip", filename_hgt)
	var cellElevationData elevationData 
	cellElevationData.data = make([]int16, CELL_SIZE*CELL_SIZE)
	cellElevationData.width = CELL_SIZE
	cellElevationData.lat = lat
	cellElevationData.lon = lon
	cellElevationData.received = false
	var swbdData []byte
	var hgtData []byte
	fmt.Printf("\nLat:%d Lon:%d\n", lat, lon)
	filename_hgt_zipped := fmt.Sprintf("terrain/%s.zip", filename_hgt)

	if !FileExists(filename_hgt_zipped) {
		fmt.Println(url)
	}

	url = fmt.Sprintf("https://e4ftl01.cr.usgs.gov/MEASURES/SRTMSWBD.003/2000.02.11/%s.zip", filename_swbd)
	filename_swbd_zipped := fmt.Sprintf("terrain/%s.zip", filename_swbd)
	if !FileExists(filename_swbd_zipped) {
		fmt.Println(url)
	} 
	if dryrun || !FileExists(filename_hgt_zipped) {
		return cellElevationData, swbdData
	}

	if FileExists(filename_hgt_zipped) {
		hgtData = unzip(filename_hgt_zipped)
	}
	if FileExists(filename_swbd_zipped) {
		swbdData = unzip(filename_swbd_zipped)
	}

	var elevation int16
	cellElevationData.received = true
	hgtBuf := bytes.NewReader(hgtData)
	for y := 0; y < CELL_SIZE; y++ {
		for x := 0; x < CELL_SIZE; x++ {
			binary.Read(hgtBuf, binary.BigEndian, &elevation)
			if swbdData[(y*3)*CELL_SWBD_SIZE+(x*3)] == 0xff {
				elevation = water_level
			}
			cellElevationData.data[y*cellElevationData.width+x] = elevation
		}
	}
	return cellElevationData, swbdData

}
func degreeMap(dryrun bool) {
	var mapDomain mapRectangle = area
	var width int = int(math.Floor((mapDomain.East - mapDomain.West) * float64(CELL_SIZE)))
	var height int = int(math.Floor((mapDomain.North - mapDomain.South) * float64(CELL_SIZE)))
	lm = newLargeMap(area, width, height)

	for lat := int16(math.Floor(mapDomain.South)); float64(lat) < mapDomain.North; lat++ {
		for lon := int16(math.Floor(mapDomain.West)); float64(lon) < mapDomain.East; lon++ {
			elevationData, _ := Download(lat, lon, dryrun)
			if dryrun {
				continue
			}

			var x_offset int = int(math.Floor((float64(elevationData.lon) - mapDomain.West) * float64(CELL_SIZE)))
			var y_offset int = int(math.Floor((mapDomain.North - float64(elevationData.lat) - 1) * float64(CELL_SIZE)))
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
					var elevation int16

					if elevationData.received {
						elevation = elevationData.data[y*elevationData.width+x]
					} else {
						elevation = math.MinInt16
					}
					cl := elevationToColor(elevation)
					(lm.data).SetRGBA(int(x_offset+x), int(y_offset+y), cl)
				}
			}
		}
	}
}
func bilinearElevation(O_value, X_value, Y_value, XY_value int16, dx, dy float64) int16 {
	return int16(math.Floor(0.5 + (1-dy)*((1-dx)*float64(O_value)+dx*float64(X_value)) + dy*((1-dx)*float64(Y_value)+dx*float64(XY_value))))
}
func latToW(deg float64) float64 {
	return math.Atanh(math.Sin(deg * math.Pi / 180))
}
func lonToV(deg float64) float64 {
	return deg * math.Pi / 180
}
func vToLon(x float64) float64 {
	return 180 / math.Pi * x
}
func wToLat(y float64) float64 {
	return math.Asin(math.Tanh(y)) * 180 / math.Pi
}
func intMin(a, b int) int {
	if a > b {
		return b
	}
	return a
}
func intMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}


func mercatorMap(pixelsize float64, base_lat float64, dryrun bool) {

	var mapDomain mapRectangle = area
	dv := lonToV(area.East) - lonToV(area.West)
	dw := latToW(area.North) - latToW(area.South)
	//	image_v_west := lonToV(area.West)
	//	image_w_north := latToW(area.North)
	if base_lat < area.South || base_lat > area.North {
		base_lat = (area.South + area.North) / 2
	}

	real_length_width := EARTH_RADIUS * math.Cos(base_lat*math.Pi/180) * ((area.East - area.West) * math.Pi / 180)
	scale := math.Floor(0.5+real_length_width/pixelsize) / dv
	width := int(math.Floor(0.5 + dv*scale))
	height := int(math.Floor(0.5 + dw*scale))

	println("Mercator width,height:",width, height)
	lm = newLargeMap(area, width, height)

	for lat := int16(math.Floor(mapDomain.South)); float64(lat) < mapDomain.North; lat++ {
		for lon := int16(math.Floor(mapDomain.West)); float64(lon) < mapDomain.East; lon++ {
			elevationData, swbdData := Download(lat, lon, dryrun)
			
			if dryrun {
				continue
			}

			cell_lon_west := math.Max(float64(lon), area.West)
			cell_lon_east := math.Min(float64(lon+1), area.East)
			cell_lat_north := math.Min(float64(lat+1), area.North)
			cell_lat_south := math.Max(float64(lat), area.South)

			// x and y are coordinate on output image
			pixel_x_offset := int( math.Ceil( (lonToV(cell_lon_west) - lonToV(area.West)) * scale) )
			pixel_x_max := int((lonToV(cell_lon_east) - lonToV(area.West)) * scale)
			pixel_y_offset := int(math.Ceil( (latToW(area.North) - latToW(cell_lat_north)) * scale) )
			pixel_y_max := int(((latToW(area.North) - latToW(cell_lat_south)) * scale) )
			println("Cell:x", pixel_x_offset, pixel_x_max, "y", pixel_y_offset, pixel_y_max)
			var pixel_lon_decimal, pixel_lat_decimal, cell_O_lon_decimal, cell_O_lat_decimal, cell_dx, cell_dy float64
			var cell_O_x, cell_O_y, cell_X_x, cell_Y_y int
			var cell_swbd_x, cell_swbd_y int
			var elevation_O, elevation_X, elevation_Y, elevation_XY, elevation int16

			for pixel_y := pixel_y_offset; pixel_y <= pixel_y_max; pixel_y++ {
				for pixel_x := pixel_x_offset; pixel_x <= pixel_x_max; pixel_x++ {

					if elevationData.received == true {
						pixel_lon_decimal = vToLon(float64(pixel_x)/scale)+area.West - math.Floor(cell_lon_west)
						pixel_lat_decimal = wToLat(latToW(area.North)-float64(pixel_y)/scale) - math.Floor(cell_lat_south)

						if pixel_lat_decimal > 1 {

							continue
						}
						if pixel_lon_decimal > 1{

							continue
						}
						//water or land
						cell_swbd_x = int(pixel_lon_decimal * float64(CELL_SWBD_DIV))
						cell_swbd_y = int((1 - pixel_lat_decimal) * float64(CELL_SWBD_DIV))
						if swbdData[cell_swbd_y*CELL_SWBD_SIZE+cell_swbd_x] == 0xff {
							elevation = water_level
						} else {
							//elevation
							cell_O_x = int(pixel_lon_decimal * float64(CELL_DIV))
							cell_O_y = int((1 - pixel_lat_decimal) * float64(CELL_DIV))
							cell_O_lon_decimal = float64(cell_O_x) / float64(CELL_DIV)
							cell_O_lat_decimal = (1 - float64(cell_O_y)/float64(CELL_DIV))
							cell_dx = (pixel_lon_decimal - cell_O_lon_decimal) * float64(CELL_DIV) // lower than 1
							cell_dy = (pixel_lat_decimal - cell_O_lat_decimal) * float64(CELL_DIV)

							if cell_dx == 0 {
								cell_X_x = cell_O_x
							} else {
								cell_X_x = cell_O_x + 1
							}
							if cell_dy == 0 {
								cell_Y_y = cell_O_y
							} else {
								cell_Y_y = cell_O_y + 1
							}

							elevation_O = elevationData.data[cell_O_y*CELL_SIZE+cell_O_x]
							elevation_X = elevationData.data[cell_O_y*CELL_SIZE+cell_X_x]
							elevation_Y = elevationData.data[cell_Y_y*CELL_SIZE+cell_O_x]
							elevation_XY = elevationData.data[cell_Y_y*CELL_SIZE+cell_X_x]
							elevation = bilinearElevation(elevation_O, elevation_X, elevation_Y, elevation_XY, cell_dx, cell_dy)
						}

					} else {
						elevation = math.MinInt16
					}

					cl := elevationToColor(elevation)
					(lm.data).SetRGBA(pixel_x, pixel_y, cl)

				}

			}
		}
	}

}
func main() {
	dryrun := flag.Bool("d", false, "check files")
	filename := flag.String("f", "default.json", "filename")
	flag.Parse()
	fmt.Println(*dryrun)
	var jsonIn jsonData
	/*
	dec := json.NewDecoder(os.Stdin)
	dec.Decode(&jsonIn)
	fmt.Printf("%+v\n", jsonIn)
    */


	json_file, err := ioutil.ReadFile(*filename)
	if err != nil{
		log.Fatalln(err)
	}
	err = json.Unmarshal(json_file,&jsonIn)
	if err != nil{
		log.Fatalln(err)
	}

	area = jsonIn.Area

	elevation_level = jsonIn.Elevation.Level // global
	water_level = jsonIn.Elevation.Water     //global

	margin_type_string := strings.ToLower(jsonIn.Drawing.Margin)
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
	drawing_type_string := strings.ToLower(jsonIn.Drawing.Style)
	switch drawing_type_string {
	case "mercator":
		drawing_style = Mercator
		mercatorMap(jsonIn.Drawing.Pixelsize, jsonIn.Drawing.Baselat, *dryrun)
	case "degree":
		drawing_style = Degree
		degreeMap(*dryrun)
	default:
		drawing_style = Degree
		degreeMap(*dryrun)
	}

	lm.SaveImageLarge(jsonIn.Filename)
}
