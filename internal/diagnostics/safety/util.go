package safety

import "os/user"

func errString(err error) string {
  if err == nil {
    return ""
  }
  return err.Error()
}

func uidOrErr(u *user.User, err error) string {
  if err != nil {
    return err.Error()
  }
  return u.Uid
}
