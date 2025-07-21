package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type Padding struct {
	top    int
	right  int
	bottom int
	left   int
}

type Config struct {
	bgColor color.Color
	padding Padding
}

func main() {
	var bgColor string
	flag.StringVar(&bgColor, "bg", "white", "Determines the background color for jpeg files")

	var padding string
	flag.StringVar(&padding, "padding", "", "Configure image padding")

	flag.Parse()

	args := flag.Args()

	if len(args) != 2 {
		log.Fatalln("must provide both input file and output file names")
	}

	inFile := args[0]
	outFile := args[1]

	fmt.Println("Converting:", inFile)

	parsedColor, err := parseBackgroundColor(bgColor)
	if err != nil {
		log.Fatalln(err)
	}

	parsedPadding, err := parsePadding(padding)
	if err != nil {
		log.Fatalln(err)
	}

	config := &Config{
		bgColor: parsedColor,
		padding: *parsedPadding,
	}

	if err := convertImage(inFile, outFile, config); err != nil {
		log.Fatalln(err)
	}

	fmt.Println("Image converted:", outFile)
}

func parsePadding(paddingStr string) (*Padding, error) {
	paddings := strings.Split(paddingStr, ",")
	pdArgs := len(paddings)

	if paddingStr == "" {
		pdArgs = 0
	}

	switch pdArgs {
	case 0:
		return &Padding{
			top:    0,
			right:  0,
			bottom: 0,
			left:   0,
		}, nil
	case 1:
		padding, err := strconv.Atoi(paddings[0])
		if err != nil {
			return nil, fmt.Errorf("parse padding: %w", err)
		}

		return &Padding{
			top:    padding,
			right:  padding,
			bottom: padding,
			left:   padding,
		}, nil
	case 2:
		ypadding, err := strconv.Atoi(paddings[0])
		if err != nil {
			return nil, fmt.Errorf("parse vertical padding: %w", err)
		}

		xpadding, err := strconv.Atoi(paddings[1])
		if err != nil {
			return nil, fmt.Errorf("parse horizontal padding: %w", err)
		}

		return &Padding{
			top:    ypadding,
			right:  xpadding,
			bottom: ypadding,
			left:   xpadding,
		}, nil
	case 4:
		tpadding, err := strconv.Atoi(paddings[0])
		if err != nil {
			return nil, fmt.Errorf("parse top padding: %w", err)
		}

		rpadding, err := strconv.Atoi(paddings[1])
		if err != nil {
			return nil, fmt.Errorf("parse right padding: %w", err)
		}

		bpadding, err := strconv.Atoi(paddings[2])
		if err != nil {
			return nil, fmt.Errorf("parse bottom padding: %w", err)
		}

		lpadding, err := strconv.Atoi(paddings[3])
		if err != nil {
			return nil, fmt.Errorf("parse left padding: %w", err)
		}

		return &Padding{
			top:    tpadding,
			right:  rpadding,
			bottom: bpadding,
			left:   lpadding,
		}, nil
	default:
		return nil, fmt.Errorf("invalid padding")
	}
}

func parseBackgroundColor(colorStr string) (color.Color, error) {
	switch strings.ToLower(colorStr) {
	case "black":
		return color.Black, nil
	case "white":
		return color.White, nil
	case "red":
		return color.RGBA{R: 255}, nil
	case "green":
		return color.RGBA{G: 255}, nil
	case "blue":
		return color.RGBA{B: 255}, nil
	default:
		return parseHexColor(colorStr)
	}
}

var hexReg = regexp.MustCompile(`\w{2}`)

func parseHexColor(hexStr string) (color.Color, error) {
	colorVals := hexReg.FindAllString(strings.TrimPrefix(hexStr, "#"), 3)

	r, err := strconv.ParseInt(colorVals[0], 16, 64)
	if err != nil {
		return nil, err
	}
	g, err := strconv.ParseInt(colorVals[1], 16, 64)
	if err != nil {
		return nil, err
	}
	b, err := strconv.ParseInt(colorVals[2], 16, 64)
	if err != nil {
		return nil, err
	}

	return color.RGBA{
		R: uint8(r),
		G: uint8(g),
		B: uint8(b),
	}, err
}

func detectFormat(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".png":
		return "png"
	case ".jpeg", ".jpg":
		return "jpeg"
	default:
		return "unknown"
	}
}

func convertImage(inputFile string, outputFile string, config *Config) error {
	inputFormat := detectFormat(inputFile)
	outputFormat := detectFormat(outputFile)

	switch {
	case inputFormat == "png" && outputFormat == "jpeg":
		return convertPNGToJPEG(inputFile, outputFile, config)
	case inputFormat == "jpeg" && outputFormat == "png":
		return convertJPEGToPNG(inputFile, outputFile, config)
	default:
		return fmt.Errorf("unsupported conversion: %s to %s", inputFormat, outputFormat)
	}
}

func convertPNGToJPEG(inputFile string, outputFile string, config *Config) error {
	f, err := os.Open(inputFile)
	if err != nil {
		return err
	}

	srcImg, err := png.Decode(f)
	if err != nil {
		return err
	}
	f.Close()

	bounds := srcImg.Bounds()

	newWidth := bounds.Dx() + config.padding.right + config.padding.left
	newHeight := bounds.Dy() + config.padding.top + config.padding.bottom
	newRect := image.Rect(0, 0, newWidth, newHeight)
	offset := image.Pt(config.padding.left, config.padding.top)

	destImg := image.NewRGBA(newRect)

	bg := image.NewUniform(config.bgColor)

	draw.Draw(destImg, newRect, bg, bounds.Min, draw.Src)
	draw.Draw(destImg, bounds.Add(offset), srcImg, bounds.Min, draw.Over)

	outFile, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0644)
	if err != nil {
		return err
	}

	return jpeg.Encode(outFile, destImg, &jpeg.Options{
		Quality: 50,
	})
}

func convertJPEGToPNG(inputFile string, outputFile string, config *Config) error {
	f, err := os.Open(inputFile)
	if err != nil {
		return err
	}

	srcImg, err := jpeg.Decode(f)
	if err != nil {
		return err
	}
	f.Close()

	outFile, err := os.OpenFile(outputFile, os.O_WRONLY|os.O_EXCL|os.O_CREATE, 0644)
	if err != nil {
		return err
	}

	bounds := srcImg.Bounds()

	minWidth := bounds.Dx() + config.padding.right + config.padding.left
	minHeight := bounds.Dy() + config.padding.top + config.padding.bottom
	newRect := image.Rect(0, 0, minWidth, minHeight)
	offset := image.Pt(config.padding.left, config.padding.top)

	destImg := image.NewRGBA(newRect)

	bg := image.NewUniform(color.Transparent)

	draw.Draw(destImg, newRect, bg, bounds.Min, draw.Src)
	draw.Draw(destImg, bounds.Add(offset), srcImg, bounds.Min, draw.Over)

	return png.Encode(outFile, destImg)
}
