// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package cmd

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/containerd/platforms"
	"github.com/distribution/reference"
	"github.com/opencontainers/go-digest"
	"unikraft.com/cloud/sdk/platform"
	"unikraft.com/x/log"

	imagespec "unikraft.com/x/image-spec"

	"unikraft.com/cli/internal/config"
	"unikraft.com/cli/internal/dockerutil"
	"unikraft.com/cli/internal/mirror"
	"unikraft.com/cli/internal/multimetro"
	"unikraft.com/cli/internal/resource"
	"unikraft.com/cli/internal/resource/cmd"
	xreference "unikraft.com/cli/internal/x/reference"
	xslices "unikraft.com/cli/internal/x/slices"
)

type ImagesCmd struct {
	cmd.ResourceCmd[ImageEntry]
	cmd.GettableResourceCmd[Image]      `set:"name=image" set:"names=images"`
	cmd.ListableResourceCmd[ImageEntry] `set:"name=image" set:"names=images"`

	Copy ImagesCopyCmd `cmd:"" help:"Copy images."`
}

type Image struct {
	Ref    ImageRef[reference.Named] `field:",short"`
	Digest digest.Digest             `field:",short"`

	Config ImageConfig `field:",embed"`

	Image imagespec.Image `field:"-" json:"image"`
}

type ImageConfig struct {
	Platform string `field:",short"`

	Entrypoint []string `field:",long"`
	Cmd        []string `field:",long"`
	Env        []string `field:",long"`

	ExposedPorts []string `field:",long"`
	Volumes      []string `field:",long"`
}

func (Image) Type() resource.Type {
	return resource.Type{
		Name:  "image",
		Names: "images",
	}
}

func (i Image) Key() string {
	return i.Ref.Reference.String()
}

func (i Image) Raw() any {
	return nil // NOTE: no platform API response associated
}

func (i Image) Fields() ([]resource.Field, error) {
	return resource.FieldsFromStruct(i)
}

func (Image) Get(ctx context.Context, keys []string) ([]resource.Resource, error) {
	opts, err := storageOpts(ctx)
	if err != nil {
		return nil, err
	}

	resources := make([]resource.Resource, 0, len(keys))
	for _, key := range keys {
		img, err := imagespec.Load(ctx, key, opts...)
		if err != nil {
			return nil, err
		}
		defer img.Close()

		config := img.Image
		resource := Image{
			Ref: ImageRef[reference.Named]{
				Reference: img.Name,
			},
			Digest: img.Descriptor.Digest,
			Image:  *img,
			Config: ImageConfig{
				Entrypoint:   config.Config.Entrypoint,
				Cmd:          config.Config.Cmd,
				Env:          config.Config.Env,
				Platform:     platforms.Format(config.Platform),
				ExposedPorts: slices.Collect(maps.Keys(config.Config.ExposedPorts)),
				Volumes:      slices.Collect(maps.Keys(config.Config.Volumes)),
			},
		}
		resources = append(resources, &resource)
	}
	return resources, nil
}

type ImageEntry struct {
	MetroName string `mirror:"metro.name" field:"metro,short"`

	Ref    ImageRef[reference.NamedTagged]   `field:",short"`
	Refs   []ImageRef[reference.NamedTagged] `field:",long"`
	Digest digest.Digest                     `field:",short"`

	Namespace string
	Dangling  bool `field:",long"`

	Canonical reference.Canonical `field:"-"`

	Image platform.Image `field:"-" json:"image"`
	Metro *config.Metro  `field:"-" json:"metro"`
}

func (ImageEntry) Type() resource.Type {
	return resource.Type{
		Name:  "image",
		Names: "images",
	}
}

func (i ImageEntry) Key() string {
	return i.Ref.Reference.String()
}

func (i ImageEntry) Raw() any {
	return i.Image
}

func (i ImageEntry) Fields() ([]resource.Field, error) {
	result, err := resource.FieldsFromStruct(i)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (ImageEntry) List(ctx context.Context) ([]resource.Resource, error) {
	cl, err := multimetro.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	resources, err := multimetro.DoAll(ctx, cl, func(ctx context.Context, mc *multimetro.MetroClient) ([]resource.Resource, error) {
		log.G(ctx).Trace().Msg("listing images")
		resp, err := mc.GetImages(ctx, platform.TagOrDigest{}, "")
		if err != nil {
			return nil, err
		}
		var results []resource.Resource
		for _, image := range resp.Data.Images {
			result, err := ImageEntry{}.load(image, &mc.Metro, false)
			if err != nil {
				return nil, err
			}
			for _, result := range result {
				results = append(results, result)
			}
		}
		return results, nil
	})
	if err != nil {
		return nil, err
	}
	return xslices.Flatten(resources), nil
}

func (ImageEntry) Get(ctx context.Context, keys []string) ([]resource.Resource, error) {
	cl, err := multimetro.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	multimetroKeys := make([]multimetro.Key, 0, len(keys))
	for _, key := range keys {
		named, err := parseNormalizedNamed(key)
		if err != nil {
			return nil, fmt.Errorf("could not parse image key %q: %w", key, err)
		}
		multimetroKeys = append(multimetroKeys, imageRefToKey(cl.Metros(), named))
	}

	resources, err := multimetro.DoKeys(ctx, cl, multimetroKeys, func(ctx context.Context, mc *multimetro.MetroClient, keys multimetro.Keys) ([]resource.Resource, []multimetro.Key, error) {
		log.G(ctx).Trace().Msg("getting images")
		resp, err := mc.GetImages(ctx, platform.TagOrDigest{}, "")
		if err != nil && !platform.ErrorContainsOnly(err, platform.APIHTTPErrorNotFound) {
			return nil, nil, err
		}
		var found []multimetro.Key
		var results []resource.Resource
		for _, image := range resp.Data.Images {
			result, err := ImageEntry{}.load(image, &mc.Metro, true)
			if err != nil {
				return nil, nil, err
			}
			for _, key := range keys {
				for _, result := range result {
					if xreference.MatchNamed(result.Canonical, key.Name) {
						found = append(found, key)
						results = append(results, result)
						break
					}
				}
			}
		}
		return results, found, nil
	})
	if err != nil {
		return nil, err
	}
	return resources, nil
}

func (ImageEntry) load(image platform.Image, metro *config.Metro, allowDangling bool) ([]ImageEntry, error) {
	if image.Digest == nil {
		return nil, fmt.Errorf("image has no digest")
	}
	base, err := parseNormalizedNamedMetro(metro, *image.Digest)
	if err != nil {
		return nil, fmt.Errorf("could not parse image ref %q: %w", *image.Digest, err)
	}
	baseDigested, ok := base.(reference.Digested)
	if !ok {
		return nil, fmt.Errorf("image ref %q is not digested", *image.Digest)
	}
	base = reference.TrimNamed(base)

	if len(image.Tags) == 0 && allowDangling {
		// Allow for dangling images (images with no tags)
		ref, err := reference.WithTag(base, "latest")
		if err != nil {
			return nil, fmt.Errorf("could not create dangling image tag: %w", err)
		}
		canonical, err := reference.WithDigest(ref, baseDigested.Digest())
		if err != nil {
			return nil, fmt.Errorf("could not create dangling image canonical reference: %w", err)
		}

		result := ImageEntry{
			Image:     image,
			Metro:     metro,
			Canonical: canonical,
			Digest:    baseDigested.Digest(),
			Dangling:  true,
		}
		err = mirror.Mirror(result, &result)
		if err != nil {
			return nil, fmt.Errorf("could not mirror image data: %w", err)
		}

		result.Ref.Reference = ref
		if ns, _, ok := strings.Cut(reference.Path(ref), "/"); ok {
			result.Namespace = ns
		}
		return []ImageEntry{result}, nil
	}

	tagged := make([]reference.NamedTagged, 0, len(image.Tags))
	for _, tag := range image.Tags {
		_, tag, ok := strings.Cut(tag, ":")
		if !ok {
			return nil, fmt.Errorf("could not parse image tag %q", tag)
		}

		ref, err := reference.WithTag(base, tag)
		if err != nil {
			return nil, fmt.Errorf("could not parse image tag %q: %w", tag, err)
		}
		tagged = append(tagged, ref)
	}

	// move latest to front if present
	idx := slices.IndexFunc(tagged, func(t reference.NamedTagged) bool {
		return t.Tag() == "latest"
	})
	if idx > 0 {
		latest := tagged[idx]
		tagged = append(tagged[:idx], tagged[idx+1:]...)
		tagged = append([]reference.NamedTagged{latest}, tagged...)
	}

	results := make([]ImageEntry, 0, len(image.Tags))
	for _, tag := range tagged {
		canonical, err := reference.WithDigest(tag, baseDigested.Digest())
		if err != nil {
			return nil, fmt.Errorf("could not create dangling image canonical reference: %w", err)
		}

		result := ImageEntry{
			Image:     image,
			Metro:     metro,
			Canonical: canonical,
			Digest:    baseDigested.Digest(),
		}
		err = mirror.Mirror(result, &result)
		if err != nil {
			return nil, fmt.Errorf("could not mirror image data: %w", err)
		}

		result.Ref.Reference = tag
		if ns, _, ok := strings.Cut(reference.Path(tag), "/"); ok {
			result.Namespace = ns
		}
		for _, t := range tagged {
			result.Refs = append(result.Refs, ImageRef[reference.NamedTagged]{
				Reference: t,
			})
		}

		results = append(results, result)
	}

	return results, nil
}

func parseNormalizedNamed(key string) (reference.Named, error) {
	return parseNormalizedNamedMetro(nil, key)
}

func parseNormalizedNamedMetro(metro *config.Metro, key string) (reference.Named, error) {
	index := "index.unikraft.io"
	if metro != nil {
		index = metro.Index()
	}
	return xreference.ParseNormalizedNamed(
		key,
		xreference.WithDefaultDomain(index),
		xreference.WithDefaultPrefix("official"),
	)
}

func imageRefToKey(metros []config.Metro, named reference.Named) multimetro.Key {
	domain := reference.Domain(named)
	if domain == "index.unikraft.io/" {
		return multimetro.Key{
			Name: named.String(),
		}
	}
	for _, metro := range metros {
		if domain == metro.Index() {
			return multimetro.Key{
				Metro: metro.Name,
				Name:  named.String(),
			}
		}
	}
	return multimetro.Key{
		Name: named.String(),
	}
}

type ImagesCopyCmd struct {
	Source string `arg:"" help:"Source image reference."`
	Dest   string `arg:"" help:"Destination image reference."`
}

func (cmd ImagesCopyCmd) Run(ctx context.Context) error {
	opts, err := storageOpts(ctx)
	if err != nil {
		return fmt.Errorf("getting storage options: %w", err)
	}

	img, err := imagespec.Load(ctx, cmd.Source, opts...)
	if err != nil {
		return fmt.Errorf("loading image from source: %w", err)
	}
	defer img.Close()

	err = imagespec.Save(ctx, cmd.Dest, img, opts...)
	if err != nil {
		return fmt.Errorf("saving image to destination: %w", err)
	}

	return nil
}

func storageOpts(ctx context.Context) ([]imagespec.StorageOpt, error) {
	cfg := config.FromContextOrDefault(ctx)
	profile, err := cfg.CurrentProfile()
	if err != nil {
		return nil, err
	}

	return []imagespec.StorageOpt{
		imagespec.WithResolver(dockerutil.Resolver(profile)),
		imagespec.WithReferenceParser(parseNormalizedNamed),
	}, nil
}
