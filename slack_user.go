package main

type SlackUser struct {
	ID      string
	NicName string
}

type SlackUsers []SlackUser

func (u SlackUsers) Find(id string) bool{
	for _, uid := range u{
		if uid.ID == id{
			return true
		}	
	}
	return false
}
