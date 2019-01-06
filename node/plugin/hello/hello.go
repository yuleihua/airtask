package main

import (
	"context"
	"fmt"
)

func TaskMain(ctx context.Context) error {

	fmt.Println("main")
	return nil
}

func TaskErr(ctx context.Context, err error) error {

	fmt.Println("error is ", err)
	return nil
}
