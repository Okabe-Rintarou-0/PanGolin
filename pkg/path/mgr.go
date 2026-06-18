package path

type PathManager interface {
	CurrentPath() *Path
}

func NewPathManager() PathManager {
	return &pathManager{
		currPath: NewPath(CloudDisk, "/"),
	}
}

type pathManager struct {
	currPath *Path
}

func (m *pathManager) CurrentPath() *Path {
	return m.currPath
}
