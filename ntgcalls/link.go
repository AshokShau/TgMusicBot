package ntgcalls

/*
#cgo linux LDFLAGS: -L${SRCDIR}/.. -lntgcalls -lm -lz -lresolv
#cgo darwin LDFLAGS: -L${SRCDIR}/.. -lntgcalls -lc++ -lz -lbz2 -liconv -framework AVFoundation -framework AudioToolbox -framework CoreAudio -framework QuartzCore -framework CoreMedia -framework VideoToolbox -framework AppKit -framework Metal -framework MetalKit -framework OpenGL -framework IOSurface -framework ScreenCaptureKit

// Windows currently uses a dynamically linked library.
#cgo windows LDFLAGS: -L${SRCDIR}/.. -lntgcalls
*/
import "C"
