package vips

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"strconv"

	"github.com/cshum/vipsgen/vips"
)

func matrixImageFromArray(arr []float64, size int) *vips.Image {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%d %d\n", size, size)

	for i, v := range arr {
		if i%size == 0 && i != 0 {
			buf.WriteString("\n")
		} else if i%size != 0 {
			buf.WriteString(" ")
		}
		buf.WriteString(strconv.FormatFloat(v, 'f', 4, 64))
	}
	buf.WriteString("\n")

	source := vips.NewSource(io.NopCloser(&buf))
	m, err := vips.NewMatrixloadSource(source, vips.DefaultMatrixloadSourceOptions())
	if err != nil {
		log.Fatalln(err)
	}
	return m
}

var sepiaMatrix = []float64{
	0.3588, 0.7044, 0.1368,
	0.2990, 0.5870, 0.1140,
	0.2392, 0.4696, 0.0912,
}

var vividMatrix = []float64{
	1.2, 0, 0,
	0, 1.2, 0,
	0, 0, 1.2,
}

var dystopianMatrix = []float64{
	0.8, 0.1, 0.1,
	0.0, 1.0, 0.0,
	0.1, 0.1, 0.9,
}

var filmMatrix = []float64{
	1.1, 0.05, 0,
	0.05, 1.0, 0,
	0, 0.05, 0.9,
}

var wildMatrix = []float64{
	1.5, 0, 0,
	0, 1.5, 0,
	0, 0, 1.5,
}

var (
	sepiaMatrixImage     = matrixImageFromArray(sepiaMatrix, 3)
	vividMatrixImage     = matrixImageFromArray(vividMatrix, 3)
	dystopianMatrixImage = matrixImageFromArray(dystopianMatrix, 3)
	filmMatrixImage      = matrixImageFromArray(filmMatrix, 3)
	wildMatrixImage      = matrixImageFromArray(wildMatrix, 3)
)
