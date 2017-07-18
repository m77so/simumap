package main

import (
	"fmt"
	"os"


	"image"
	"image/color"

	"log"

	"image/png"

)


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
func main() {
	img := image.NewRGBA(image.Rect(0,0,255*3,20))
	for x:=0;x<20;x++{
		for ca:=0;ca<=255;ca++{
			c := uint8(ca)

			cl := color.RGBA{c,c,c,255}
			img.SetRGBA(ca*3,x,cl)
			img.SetRGBA(ca*3+1,x,cl)
			img.SetRGBA(ca*3+2,x,cl)

		}
	}
	saveImage(img,"sub.png")

}
