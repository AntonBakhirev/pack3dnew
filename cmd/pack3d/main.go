package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/fogleman/fauxgl"
	"github.com/fogleman/pack3d/pack3d"
)

const (
	offset = 0
)

func timed(name string) func() {
	if len(name) > 0 {
		fmt.Printf("%s... ", name)
	}
	start := time.Now()
	return func() {
		fmt.Println(time.Since(start))
	}
}

func main() {
	var done func()

	bvhDetail := 12
	annealingIterations := 2000000
	LockX := false
	LockY := false
	LockZ := false
	AngleX := int(90)
	AngleY := int(90)
	AngleZ := int(90)

	rand.Seed(time.Now().UTC().UnixNano())

	for _, arg := range os.Args[1:] {
		if arg == "-lx" {
			LockX = true
			AngleX = 360
		}

		if arg == "-ly" {
			LockY = true
			AngleY = 360
		}

		if arg == "-lz" {
			LockZ = true
			AngleZ = 360
		}

		if len(arg) >= 3 {

			if arg[:2] == "-x" {
				f, err := strconv.ParseInt(arg[2:], 10, 64)
				if err == nil {
					AngleX = int(f)
				}
			}

			if arg[:2] == "-y" {
				f, err := strconv.ParseInt(arg[2:], 10, 64)
				if err == nil {
					AngleY = int(f)
				}
			}

			if arg[:2] == "-z" {
				f, err := strconv.ParseInt(arg[2:], 10, 64)
				if err == nil {
					AngleZ = int(f)
				}
			}

			if arg[:2] == "-d" {
				f, err := strconv.ParseInt(arg[2:], 10, 64)
				if err == nil {
					bvhDetail = int(f)
				}
			}
		}
	}

	model := pack3d.NewModel()
	pack3d.CreateRotations(LockX, LockY, LockZ, AngleX, AngleY, AngleZ)

	count := 1
	ok := false
	var totalVolume float64
	nocmd := false
	for _, arg := range os.Args[1:] {
		nocmd = true
		_count, err := strconv.ParseInt(arg, 0, 0)
		if err == nil {
			count = int(_count)
			nocmd = false
			continue
		}

		if arg == "-lx" {
			nocmd = false
			continue
		}

		if arg == "-ly" {
			nocmd = false
			continue
		}

		if arg == "-lz" {
			nocmd = false
			continue
		}

		if len(arg) >= 3 {
			if arg[:2] == "-x" {
				nocmd = false
				continue
			}

			if arg[:2] == "-y" {
				nocmd = false
				continue
			}

			if arg[:2] == "-z" {
				nocmd = false
				continue
			}

			if arg[:2] == "-d" {
				nocmd = false
				continue
			}
		}

		if nocmd == true {

			done = timed(fmt.Sprintf("loading mesh %s", arg))
			mesh, err := fauxgl.LoadMesh(arg)
			if err != nil {
				panic(err)
			}
			done()

			totalVolume += mesh.BoundingBox().Volume()
			size := mesh.BoundingBox().Size()
			fmt.Printf("  %d triangles\n", len(mesh.Triangles))
			fmt.Printf("  %g x %g x %g\n", size.X, size.Y, size.Z)

			done = timed("centering mesh")
			mesh.Center()
			done()

			done = timed("building bvh tree")
			model.Add(mesh, bvhDetail, count)
			ok = true
			done()
		}
	}

	if !ok {
		fmt.Println("Usage: [-lx] [-ly] [-lz] [-x<degree>] [-y<degree>] [-z<degree>] [-d<volume>] pack3d N1 mesh1.stl N2 mesh2.stl ...")
		fmt.Println("  Packs N copies of each mesh into as small of a volume as possible.")
		fmt.Println("  Runs forever, looking for the best packing.")
		fmt.Println("  Results are written to disk whenever a new best is found.")
		fmt.Println("  [-lx] [-ly] [-lz] locks rotation.")
		fmt.Println("  [-x<degree>] [-y<degree>] [-z<degree>] rotation step (90 is default).")
		fmt.Println("  [-d<volume>] finesse (12 is default). More is better but consumes more RAM.")
		return
	}

	side := math.Pow(totalVolume, 1.0/3)
	model.Deviation = side / 32

	best := 1e9

	out := make(chan string)
	numcpu := runtime.NumCPU()

	for ii := 0; ii < numcpu; ii++ {
		go func() {
			for {
				model = model.Pack(annealingIterations, LockX, LockY, LockZ, AngleX, AngleY, AngleZ, nil)
				score := model.Energy()
				if score < best {
					best = score
					done = timed("writing mesh")
					model.Mesh().SaveSTL(fmt.Sprintf("pack3d-%.3f.stl", score))
					// model.TreeMesh().SaveSTL(fmt.Sprintf("out%dtree.stl", int(score*100000)))
					done()
				}
				model.Reset()
			}
			out <- "done"
		}()
	}

	for i := 0; i < 32; i++ {
		<-out
	}
}
