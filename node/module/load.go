package main

import (
	"context"
	"flag"
	"fmt"
	"plugin"
)

var module string

func init() {
	flag.StringVar(&module, "m", "hello/hello.so", "module file")
}

func main() {
	flag.Parse()

	ctx := context.Background()
	//加载动态库
	p, err := plugin.Open(module)
	if err != nil {
		panic(err)
	}

	// err handle
	errHandle, err := p.Lookup("Err")
	if err != nil {
		panic(err)
	}

	// prev handle
	prev, err := p.Lookup("Prev")
	if err != nil {
		panic(err)
	}

	if err := prev.(func(ctx context.Context) error)(ctx); err != nil {
		fmt.Println("prev", err)
		errHandle.(func(ctx context.Context, err error) error)(ctx, err)
		return
	}

	main, err := p.Lookup("Main")
	if err != nil {
		panic(err)
	}

	if err := main.(func(ctx context.Context) error)(ctx); err != nil {
		fmt.Println("main", err)
		errHandle.(func(ctx context.Context, err error) error)(ctx, err)
		return
	}

	end, err := p.Lookup("End")
	if err != nil {
		panic(err)
	}

	if err := end.(func(ctx context.Context) error)(ctx); err != nil {
		fmt.Println("end", err)
		errHandle.(func(ctx context.Context, err error) error)(ctx, err)
		return
	}
	fmt.Println(">> run all function")
}
