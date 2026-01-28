package safety

import "os/user"

func checkExecutionUser() InvariantResult {
  u, err := user.Current()
  passed := err == nil && u.Uid != "0"

  return InvariantResult{
    Name:     "execution_user",
    Passed:   passed,
    Expected: "non-root user",
    Actual:   uidOrErr(u, err),
  }
}
