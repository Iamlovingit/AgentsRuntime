package gateway

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// A rollback archive records a link, not its referent. In particular a legacy
// image-owned dependency may no longer exist in the target image. Permit only
// verified managed/image roots; never follow an unknown external link.
func upgradePluginLink(name, stateRoot string) (string, error) {
	link, err := os.Readlink(name)
	if err != nil {
		return "", err
	}
	lexical := link
	if !filepath.IsAbs(lexical) {
		lexical = filepath.Join(filepath.Dir(name), lexical)
	}
	lexical = filepath.Clean(lexical)
	resolved, err := filepath.EvalSymlinks(lexical)
	if err == nil {
		if pathWithin(stateRoot, resolved) || imageOwnedPluginPath(resolved) {
			return resolved, nil
		}
		return "", fmt.Errorf("plugin link leaves managed roots: %s", name)
	}
	if !errors.Is(err, os.ErrNotExist) || !imageOwnedPluginPath(lexical) {
		return "", err
	}
	// Verify the closest existing ancestor, rejecting symlink escapes even when
	// the final file is absent. The caller only preserves the original link.
	for p := filepath.Dir(lexical); p != filepath.Dir(p); p = filepath.Dir(p) {
		parent, e := filepath.EvalSymlinks(p)
		if e == nil {
			if imageOwnedPluginPath(parent) {
				return lexical, nil
			}
			break
		}
		if !errors.Is(e, os.ErrNotExist) {
			return "", e
		}
	}
	return "", fmt.Errorf("missing image-owned link has no verified image ancestor: %s", name)
}

func imageOwnedPluginPath(path string) bool {
	return pathWithin("/defaults/.openclaw", path) || pathWithin("/usr/local/lib/node_modules/openclaw", path)
}

// Rebuild image-owned dependency links in an uninstalled candidate/probe.
// Real custom files and non-image plugins are never replaced. The complete
// original is retained by the rebuild journal before the candidate becomes live.
func rebuildImageManagedPlugins(state, defaults string) error {
	plan, err := planManagedPlugins(state, defaults)
	if err != nil {
		return err
	}
	if err := applyManagedPluginPlan(state, defaults, plan); err != nil {
		return err
	}
	for _, group := range []string{"npm/projects", "extensions"} {
		root := filepath.Join(state, filepath.FromSlash(group))
		entries, err := os.ReadDir(root)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			project := filepath.Join(root, entry.Name())
			image := filepath.Join(defaults, filepath.FromSlash(group), entry.Name())
			info, err := os.Lstat(image)
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return err
			}
			if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				continue
			}
			imageLinks := 0
			err = filepath.WalkDir(project, func(name string, e fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if e.IsDir() {
					return nil
				}
				if e.Type()&os.ModeSymlink == 0 {
					return nil
				}
				link, err := os.Readlink(name)
				if err != nil {
					return err
				}
				if !filepath.IsAbs(link) {
					link = filepath.Join(filepath.Dir(name), link)
				}
				if pathWithin(image, filepath.Clean(link)) {
					imageLinks++
					return os.Remove(name)
				}
				return nil
			})
			if err != nil {
				return err
			}
			if imageLinks == 0 {
				continue
			} // fully custom installation, not an image clone
			if err := seedImagePluginTree(image, project); err != nil {
				return err
			}
		}
	}
	return validateUpgradePluginLinks(state)
}

func seedImagePluginTree(image, dest string) error {
	return seedImagePluginTreeWithin(image, dest, "")
}

func seedImagePluginTreeWithin(image, dest, approvedRoot string) error {
	if approvedRoot != "" {
		if real, err := filepath.EvalSymlinks(approvedRoot); err == nil {
			approvedRoot = real
		}
	}
	approved := func(path string) bool {
		return imageOwnedPluginPath(path) || (approvedRoot != "" && pathWithin(approvedRoot, path))
	}
	return filepath.WalkDir(image, func(name string, e fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(image, name)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if old, err := os.Lstat(target); err == nil {
			if e.IsDir() {
				if old.Mode()&os.ModeSymlink != 0 {
					return filepath.SkipDir
				}
				if !old.IsDir() {
					return fmt.Errorf("custom plugin file conflicts with image directory: %s", target)
				}
				// A dependency directory emptied of old image links can use one
				// immutable tree link, rather than copying thousands of NFS files.
				if target != dest {
					children, err := os.ReadDir(target)
					if err != nil {
						return err
					}
					if len(children) == 0 {
						if err := os.Remove(target); err != nil {
							return err
						}
						resolved, err := filepath.EvalSymlinks(name)
						if err != nil {
							return err
						}
						if !approved(resolved) {
							return ErrWorkspacePath
						}
						if err := os.Symlink(resolved, target); err != nil {
							return err
						}
						return filepath.SkipDir
					}
				}
				return nil // merge children of an existing nonempty directory
			} else {
				return nil
			} // user-owned override, still checked by core validation
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if e.IsDir() {
			if target != dest {
				resolved, err := filepath.EvalSymlinks(name)
				if err != nil {
					return err
				}
				if !approved(resolved) {
					return ErrWorkspacePath
				}
				if err := os.Symlink(resolved, target); err != nil {
					return err
				}
				return filepath.SkipDir
			}
			return os.MkdirAll(target, 0755)
		}
		resolved, err := filepath.EvalSymlinks(name)
		if err != nil {
			return err
		}
		if !approved(resolved) {
			return fmt.Errorf("image plugin payload leaves immutable roots: %s", name)
		}
		return os.Symlink(resolved, target)
	})
}

func validateUpgradePluginLinks(state string) error {
	for _, root := range []string{"npm", "extensions", "plugin-skills"} {
		base := filepath.Join(state, root)
		if _, err := os.Lstat(base); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return err
		}
		if err := filepath.WalkDir(base, func(name string, e fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if e.Type()&os.ModeSymlink == 0 {
				return nil
			}
			resolved, err := filepath.EvalSymlinks(name)
			if err != nil {
				return fmt.Errorf("OPENCLAW_PLUGIN_REBUILD_REQUIRED: plugin dependency is unavailable at %s; retain data and rebuild using a compatible image: %w", strings.TrimPrefix(name, state+string(filepath.Separator)), err)
			}
			if !pathWithin(state, resolved) && !imageOwnedPluginPath(resolved) {
				return fmt.Errorf("plugin dependency leaves managed roots: %s", name)
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}
