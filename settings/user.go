package settings

type User struct {
	ID       string
	NickName string
}

type Users []User

func (u Users) Find(id string) bool {
	if id == "" {
		return false
	}
	for _, uid := range u {
		if uid.ID == id {
			return true
		}
	}
	return false
}
