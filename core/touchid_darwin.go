//go:build darwin

package core

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework LocalAuthentication -framework Foundation
#import <LocalAuthentication/LocalAuthentication.h>

int nsh_authenticate_touch_id(const char *reason) {
    __block int result = 0;
    dispatch_semaphore_t sema = dispatch_semaphore_create(0);

    LAContext *ctx = [[LAContext alloc] init];
    NSError *authError = nil;
    NSString *nsReason = [NSString stringWithUTF8String:reason];

    if ([ctx canEvaluatePolicy:LAPolicyDeviceOwnerAuthenticationWithBiometrics error:&authError]) {
        [ctx evaluatePolicy:LAPolicyDeviceOwnerAuthenticationWithBiometrics
            localizedReason:nsReason
                      reply:^(BOOL success, NSError *error) {
            result = success ? 1 : 0;
            dispatch_semaphore_signal(sema);
        }];
        dispatch_semaphore_wait(sema, DISPATCH_TIME_FOREVER);
    } else {
        // Fallback to device passcode (password)
        if ([ctx canEvaluatePolicy:LAPolicyDeviceOwnerAuthentication error:&authError]) {
            [ctx evaluatePolicy:LAPolicyDeviceOwnerAuthentication
                localizedReason:nsReason
                          reply:^(BOOL success, NSError *error) {
                result = success ? 1 : 0;
                dispatch_semaphore_signal(sema);
            }];
            dispatch_semaphore_wait(sema, DISPATCH_TIME_FOREVER);
        }
    }
    return result;
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// AuthenticateTouchID prompts for Touch ID (or falls back to system password)
func AuthenticateTouchID(reason string) error {
	cReason := C.CString(reason)
	defer C.free(unsafe.Pointer(cReason))

	result := C.nsh_authenticate_touch_id(cReason)
	if result != 1 {
		return fmt.Errorf("authentication failed")
	}
	return nil
}
