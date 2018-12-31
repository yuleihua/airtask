package main

import (
	"context"
	"fmt"
)

func Prev(ctx context.Context) error {

	fmt.Printf("prev")
	return nil
}

func Main(ctx context.Context) error {

	fmt.Printf("main")
	return nil
}

func End(ctx context.Context) error {

	fmt.Printf("end")
	return nil
}

func Err(ctx context.Context, err error) error {

	fmt.Printf("handle err is %v", err)
	return nil
}
