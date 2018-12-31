package main

import (
	"context"
	"fmt"
)

func Prev(ctx context.Context) error {

	fmt.Println("prev")
	return nil
}

func Main(ctx context.Context) error {

	fmt.Println("main")
	return nil
}

func End(ctx context.Context) error {

	fmt.Println("end")
	return nil
}

func Err(ctx context.Context, err error) error {

	fmt.Println("error is ", err)
	return nil
}
