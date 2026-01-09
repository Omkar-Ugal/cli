// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package cmd

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/distribution/reference"
	"github.com/opencontainers/go-digest"
	"unikraft.com/cloud/sdk/platform"
	"unikraft.com/x/log"

	"unikraft.com/cli/internal/config"
	"unikraft.com/cli/internal/mirror"
	"unikraft.com/cli/internal/multimetro"
	"unikraft.com/cli/internal/resource"
	xslices "unikraft.com/cli/internal/x/slices"
)

type ImagesCmd struct {
	resource.ResourceCmd[Image] `set:"name=image" set:"names=images"`
}

type Image struct {
	MetroName string `mirror:"metro.name" field:"metro,short"`

	Ref    ImageRef[reference.NamedTagged]   `field:",short"`
	Refs   []ImageRef[reference.NamedTagged] `field:",long"`
	Digest digest.Digest                     `field:",short"`

	Namespace string
	Dangling  bool `field:",long"`

	Image platform.Image `field:"-" json:"image"`
	Metro *config.Metro  `field:"-" json:"metro"`
}

func (Image) Type() resource.Type {
	return resource.Type{
		Name:  "image",
		Names: "images",
	}
}

func (i Image) key() multimetro.Key {
	return multimetro.Key{
		Metro: i.Metro.Name,
		Name:  i.Ref.String(),
	}
}

func (i Image) Key() string {
	return i.key().String()
}

func (i Image) Raw() any {
	return i.Image
}

func (i Image) Fields() ([]resource.Field, error) {
	result, err := resource.FieldsFromStruct(i)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (Image) List(ctx context.Context) ([]resource.Resource, error) {
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
			result, err := Image{}.load(image, &mc.Metro, false)
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

func (Image) Get(ctx context.Context, keys []string) ([]resource.Resource, error) {
	cl, err := multimetro.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	resources, err := multimetro.DoKeys(ctx, cl, multimetro.ParseKeys(keys), func(ctx context.Context, mc *multimetro.MetroClient, keys multimetro.Keys) ([]resource.Resource, []multimetro.Key, error) {
		log.G(ctx).Trace().Msg("getting images")
		resp, err := mc.GetImages(ctx, platform.TagOrDigest{}, "")
		if err != nil && !platform.ErrorContainsOnly(err, platform.APIHTTPErrorNotFound) {
			return nil, nil, err
		}
		var found []multimetro.Key
		var results []resource.Resource
		for _, image := range resp.Data.Images {
			result, err := Image{}.load(image, &mc.Metro, true)
			if err != nil {
				return nil, nil, err
			}
			for _, key := range keys {
				for _, result := range result {
					if result.matches(key.Name) {
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

func (i Image) matches(target string) bool {
	named, err := reference.ParseNormalizedNamed(target)
	if err != nil {
		return false
	}
	named = reference.TagNameOnly(named)

	ref := i.Ref.Reference
	if named.Name() != ref.Name() {
		return false
	}
	if i.Dangling {
		if _, ok := named.(reference.Digested); !ok {
			return false
		}
	}

	if digestRef, ok := named.(reference.Digested); ok {
		if i.Digest != digestRef.Digest() {
			return false
		}
	} else if tagged, ok := named.(reference.Tagged); ok {
		if ref.Tag() != tagged.Tag() {
			return false
		}
	}

	return true
}

func (Image) load(image platform.Image, metro *config.Metro, allowDangling bool) ([]Image, error) {
	if image.Digest == nil {
		return nil, fmt.Errorf("image has no digest")
	}
	base, err := reference.ParseNormalizedNamed(*image.Digest)
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

		result := Image{
			Image: image,
			Metro: metro,
		}
		err = mirror.Mirror(result, &result)
		if err != nil {
			return nil, fmt.Errorf("could not mirror image data: %w", err)
		}

		result.Dangling = true
		result.Digest = baseDigested.Digest()
		result.Ref.Reference = ref
		if ns, _, ok := strings.Cut(reference.Path(ref), "/"); ok {
			result.Namespace = ns
		}
		return []Image{result}, nil
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

	results := make([]Image, 0, len(image.Tags))
	for _, tag := range tagged {
		result := Image{
			Image: image,
			Metro: metro,
		}
		err := mirror.Mirror(result, &result)
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
		result.Digest = baseDigested.Digest()

		results = append(results, result)
	}

	return results, nil
}
