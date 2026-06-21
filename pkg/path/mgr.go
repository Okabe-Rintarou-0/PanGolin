package path

 import stdpath "path"

type PathManager interface {
	CurrentPath() *Path
 	ChangeDir(target string) error
}

func NewPathManager() PathManager {
	return &pathManager{
		currPath: NewPath(CloudDisk, "/", true),
	}
}

type pathManager struct {
	currPath *Path
}

func (m *pathManager) CurrentPath() *Path {
	return m.currPath
}
 
 func (m *pathManager) ChangeDir(target string) error {
 	curr := m.currPath.Path()
 
	var newPath string
	if target == "" || target == "~" {
		newPath = "/"
	} else if target[0] == '/' {
 		newPath = stdpath.Clean(target)
	} else {
 		newPath = stdpath.Join(curr, target)
	}

	m.currPath = NewPath(m.currPath.Device(), newPath, true)
 	return nil
 }
