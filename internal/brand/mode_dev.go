//go:build !production

package brand

// Dev reports whether this is a development build. Production builds are
// compiled with `-tags production` by the Taskfile.
const Dev = true
