// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package term

import "syscall"

const ioctlReadTermios = syscall.TIOCGETA

// Termios functions describe a general terminal interface that is
// provided to control asynchronous communications ports.
type Termios syscall.Termios
