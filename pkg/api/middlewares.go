/*
 * Copyright (c) 2017, MegaEase
 * All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package api

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/megaease/easegress/pkg/common"
	"github.com/megaease/easegress/pkg/logger"

	"github.com/kataras/iris/context"
)

func newAPILogger() func(context.Context) {
	return func(ctx context.Context) {
		var (
			method            string
			remoteAddr        string
			path              string
			code              int
			bodyBytesReceived int64
			bodyBytesSent     int64
			startTime         time.Time
			processTime       time.Duration
		)

		startTime = common.Now()
		ctx.Next()
		processTime = common.Now().Sub(startTime)

		method = ctx.Method()
		remoteAddr = ctx.RemoteAddr()
		path = ctx.Path()
		code = ctx.GetStatusCode()
		bodyBytesReceived = ctx.GetContentLength()
		bodyBytesSent = int64(ctx.ResponseWriter().Written())

		logger.APIAccess(method, remoteAddr, path, code,
			bodyBytesReceived, bodyBytesSent,
			startTime, processTime)
	}
}

func newRecoverer() func(context.Context) {
	return func(ctx context.Context) {
		defer func() {
			if err := recover(); err != nil {
				if ctx.IsStopped() {
					return
				}

				logger.Errorf("recover from %s, err: %v, stack trace:\n%s\n",
					ctx.HandlerName(), err, debug.Stack())
				if ce, ok := err.(clusterErr); ok {
					HandleAPIError(ctx, http.StatusServiceUnavailable, ce)
				} else {
					HandleAPIError(ctx, http.StatusInternalServerError, fmt.Errorf("%v", err))
				}
			}
		}()

		ctx.Next()
	}
}

func newConfigVersionAttacher(s *Server) func(context.Context) {
	return func(ctx context.Context) {
		// NOTE: It needs to add the header before the next handlers
		// write the body to the network.
		version := s._getVersion()
		ctx.ResponseWriter().Header().Set(ConfigVersionKey, fmt.Sprintf("%d", version))
		ctx.Next()
	}
}
