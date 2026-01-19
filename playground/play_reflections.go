package main

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type Directory struct {
	Id        uuid.UUID `json:"id" db:"id"`
	Path      string    `json:"path" db:"path"`
	IsDone    bool      `json:"isDone" db:"is_done"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

func Display(name string, x any) {
	d := fmt.Sprintf("Display %s (%T):\n", name, x)
	io.WriteString(os.Stdout, d)
	display(name, reflect.ValueOf(x))
}

func display(path string, v reflect.Value) {
	switch v.Kind() {
	case reflect.Invalid:
		io.WriteString(os.Stdout, path+" = invalid\n")
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			f := v.Type().Field(i).Name
			display(fmt.Sprintf("%s.%s", path, f), v.Field(i))
		}
	default: // podstawowe typy, kanały, funkcje
		fmt.Printf("%s = %s\n", path, formatAtom(v))
	}
}

func main() {
	defer fmt.Println((runtime.NumGoroutine()))
	u, _ := uuid.Parse("000fb20a-c830-495a-95aa-60f593e09d19")
	t := time.Now()
	direction := Directory{
		Id:        u,
		Path:      "Z:\\ORANGE\\210816_Orange_swiatlowod_BBS\\3d\\OUTPUT\\106\\MAIN",
		IsDone:    false,
		CreatedAt: t,
		UpdatedAt: t,
	}
	// fmt.Printf("%#v", d)
	fmt.Println()

	Display("direction", direction)

	counter := 10
	ch := make(chan int, 3)
	// wg := sync.WaitGroup{}

	moment := func() {
		r := rand.Intn(500) + 200
		time.Sleep(time.Duration(r) * time.Millisecond)
		ch <- rand.Intn(1000)
	}

	for range counter {
		go moment()
	}

	fmt.Println((runtime.NumGoroutine()))

	Display("ch", ch)

	// wg.Wait()
	for range counter {
		v := <-ch
		fmt.Printf("Received value: %d\n", v)

	}

	const tickRate = time.Second
	ticker := time.NewTicker(tickRate).C

	stopper := time.After(5 * time.Second)

loop:
	for {
		select {
		case <-ticker:
			fmt.Println("Tick at", time.Now())
		case <-stopper:
			fmt.Println("Stopping ticker at", time.Now())
			break loop
		}
	}

	fmt.Println("Ticker stopped")

}

func formatAtom(v reflect.Value) string {
	switch v.Kind() {
	case reflect.Invalid:
		return "invalid"
	case reflect.Int, reflect.Int8, reflect.Int16,
		reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(v.Uint(), 10)
	// …dla zwięzłości pominięto przypadki dla liczb zmiennoprzecinkowych i zespolonych…
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.String:
		return strconv.Quote(v.String())
	case reflect.Chan, reflect.Func, reflect.Ptr, reflect.Slice, reflect.Map:
		return v.Type().String() + " 0x" +
			strconv.FormatUint(uint64(v.Pointer()), 16)
	default: // reflect.Array, reflect.Struct, reflect.Interface
		return v.Type().String() + " value"
	}
}
