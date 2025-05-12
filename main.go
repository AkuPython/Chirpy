package main

import (
	"log"
	"net/http"
)


func main()  {
	const port = "8080"
	const rootPath = "."
	
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(rootPath)))

	srv := &http.Server{
		Addr: ":" + port,
		Handler: mux,
	}
	

	log.Printf("Serving files from: '%s' on port: %s\n", rootPath, port)
	log.Fatal(srv.ListenAndServe())

}

