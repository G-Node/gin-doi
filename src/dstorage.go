package ginDoi

type LocalStorage struct {
	Path string
}

func (ls *LocalStorage) Exists(target string) (bool, error) {
	return false, nil
}

func (ls *LocalStorage) Put(source string , target string) (bool, error){
	return true, nil
}

func (ls LocalStorage) GetDataSource() (*GinDataSource, error) {
	return nil, nil
}