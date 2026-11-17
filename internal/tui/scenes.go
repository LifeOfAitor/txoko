package tui

import (
	"github.com/LifeOfAitor/txoko/internal/scene"
	"github.com/LifeOfAitor/txoko/internal/scene/bonsai"
	"github.com/LifeOfAitor/txoko/internal/scene/garden"
	"github.com/LifeOfAitor/txoko/internal/scene/island"
)

// scenes lists every available scene. The first one is the default.
var scenes = []scene.Scene{
	bonsai.Scene{},
	island.Scene{},
	garden.Scene{},
}

// SceneNames returns the names of all scenes, default first.
func SceneNames() []string {
	names := make([]string, len(scenes))
	for i, s := range scenes {
		names[i] = s.Name()
	}
	return names
}

// sceneIndex finds a scene by name. An empty name means the default scene.
func sceneIndex(name string) (int, bool) {
	if name == "" {
		return 0, true
	}
	for i, s := range scenes {
		if s.Name() == name {
			return i, true
		}
	}
	return 0, false
}
