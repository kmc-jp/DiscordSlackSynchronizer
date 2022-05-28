package settings

type User struct {
	ID       string
	NickName string
}

type Users []User

func (u Users) Find(id string) bool {
	for _, uid := range u {
		if uid.ID == id {
			return true
		}
	}
	return false
}
