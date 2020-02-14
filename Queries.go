package main

func isConnectedToDB() error {
	sqlCheckConnection := "SELECT UNIX_TIMESTAMP()"
	var count int
	err := queryRow(&count, sqlCheckConnection)
	if err != nil {
		return err
	}
	return nil
}
