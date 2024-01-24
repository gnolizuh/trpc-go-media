//
// Copyright [2024] [https://github.com/gnolizuh]
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package writev

// Options is the Buffer configuration.
type Options struct {
	handler    QuitHandler // Set the goroutine to exit the cleanup function.
	bufferSize int         // Set the length of each connection request queue.
	dropFull   bool        // Whether the queue is full or not.
}

// Option optional parameter.
type Option func(*Options)

// WithQuitHandler returns an Option which sets the Buffer goroutine exit handler.
func WithQuitHandler(handler QuitHandler) Option {
	return func(o *Options) {
		o.handler = handler
	}
}

// WithBufferSize returns an Option which sets the length of each connection request queue.
func WithBufferSize(size int) Option {
	return func(opts *Options) {
		opts.bufferSize = size
	}
}

// WithDropFull returns an Option which sets whether to drop the request when the queue is full.
func WithDropFull(drop bool) Option {
	return func(opts *Options) {
		opts.dropFull = drop
	}
}
