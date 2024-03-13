package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Running and listening HTTP server
func (app *application) serve() error {
	// HTTP server settings
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		ErrorLog:     log.New(app.logger, "", 0),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	// shutdownError channel receive any errors returned by the graceful Shutdown() function
	shutdownError := make(chan error)

	// background go routine for listening a signal
	go func() {
		// quit channel which carries os.Signal values with buffered capacity 1 because avoiding missing
		// signal when sinal.Notify() function will not block and wait until the channel is ready to receive the signal.
		quit := make(chan os.Signal, 1)

		// signal.Notify() catch SIGNAL INTERRUPT and SIGNAL TERMINATE and relay them to quit channel
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		// waiting for os.Signal() being caught
		s := <-quit

		app.logger.PrintInfo("caught signal", map[string]string{
			"signal": s.String(),
		})

		// 5-second context timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// http.Server.Shutdown() implement a graceful shutdown however limit the time server
		// waiting for shutdown with context, return nil if graceful shutdown was successful or error
		err := srv.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
		}

		app.logger.PrintInfo("completing background tasks", map[string]string{
			"addr": srv.Addr,
		})

		// waiting for background goroutine complete
		app.wg.Wait()
		shutdownError <- nil
	}()

	app.logger.PrintInfo("starting server", map[string]string{
		"addr": srv.Addr,
		"env":  app.config.env,
	})

	// calling Shutdown() will cause ListenAndServe() to immediately return a http.ErrServerClosed error.
	// That indicate graceful shutdown has started, if not http.ErrServerClosed error, something failed and return that error
	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	// waiting for graceful shutdown done and receive the error on Shutdown()
	err = <-shutdownError
	if err != nil {
		return err
	}

	// graceful shutdown completed SUCCESSFULLY
	app.logger.PrintInfo("stopped server", map[string]string{
		"addr": srv.Addr,
	})

	return nil
}
