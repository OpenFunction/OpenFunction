/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// +kubebuilder:docs-gen:collapse=Apache License

package v1beta1

/*
   For imports, we'll need the controller-runtime
   [`conversion`](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/conversion?tab=doc)
   package, plus the API version for our hub type (v1), and finally some of the
   standard packages.
*/
import (
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	"github.com/openfunction/apis/core/v1beta2"
)

// +kubebuilder:docs-gen:collapse=Imports

/*
Our "spoke" versions need to implement the
[`Convertible`](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/conversion?tab=doc#Convertible)
interface.  Namely, they'll need `ConvertTo` and `ConvertFrom` methods to convert to/from
the hub version.
*/

/*
ConvertTo is expected to modify its argument to contain the converted object.
Most of the conversion is straightforward copying, except for converting our changed field.
*/
// ConvertTo converts this Function to the Hub version (v1beta2).
func (src *Builder) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1beta2.Builder)
	dst.ObjectMeta = src.ObjectMeta

	dst.Spec.Builder = src.Spec.Builder
	dst.Spec.Dockerfile = src.Spec.Dockerfile
	dst.Spec.Timeout = src.Spec.Timeout
	dst.Spec.Image = src.Spec.Image
	dst.Spec.ImageCredentials = src.Spec.ImageCredentials
	dst.Spec.BuilderCredentials = src.Spec.BuilderCredentials
	dst.Spec.Env = src.Spec.Env
	dst.Spec.State = v1beta2.BuilderState(src.Spec.State)

	if src.Spec.SrcRepo != nil {
		dst.Spec.SrcRepo = &v1beta2.GitRepo{
			Url:           src.Spec.SrcRepo.Url,
			Revision:      src.Spec.SrcRepo.Revision,
			SourceSubPath: src.Spec.SrcRepo.SourceSubPath,
			Credentials:   src.Spec.SrcRepo.Credentials,
		}

		if src.Spec.SrcRepo.BundleContainer != nil {
			dst.Spec.SrcRepo.BundleContainer = &v1beta2.BundleContainer{
				Image: src.Spec.SrcRepo.BundleContainer.Image,
			}
		}
	}

	if src.Spec.Shipwright != nil {
		dst.Spec.Shipwright = &v1beta2.ShipwrightEngine{
			Timeout: src.Spec.Shipwright.Timeout,
		}

		if src.Spec.Shipwright.Strategy != nil {
			dst.Spec.Shipwright.Strategy = &v1beta2.Strategy{
				Name: src.Spec.Shipwright.Strategy.Name,
				Kind: src.Spec.Shipwright.Strategy.Kind,
			}
		}

		if src.Spec.Params != nil {
			for k, v := range src.Spec.Params {
				value := v
				dst.Spec.Shipwright.Params = append(dst.Spec.Shipwright.Params, &v1beta2.ParamValue{
					SingleValue: &v1beta2.SingleValue{
						Value: &value,
					},
					Name: k,
				})
			}
		}
	}

	// Status
	dst.Status.State = src.Status.State
	dst.Status.Phase = src.Status.Phase
	dst.Status.ResourceRef = src.Status.ResourceRef
	dst.Status.Reason = src.Status.Reason

	if src.Status.Output != nil {
		dst.Status.Output = &v1beta2.BuilderOutput{
			Digest: src.Status.Output.Digest,
			Size:   src.Status.Output.Size,
		}
	}

	if src.Status.Sources != nil {
		for _, item := range src.Status.Sources {
			res := v1beta2.SourceResult{
				Name: item.Name,
			}

			if item.Git != nil {
				res.Git = &v1beta2.GitSourceResult{
					CommitSha:    item.Git.CommitSha,
					CommitAuthor: item.Git.CommitAuthor,
					BranchName:   item.Git.BranchName,
				}
			}

			if item.Bundle != nil {
				res.Bundle = &v1beta2.BundleSourceResult{
					Digest: item.Bundle.Digest,
				}
			}

			dst.Status.Sources = append(dst.Status.Sources, res)
		}
	}

	return nil
}

/*
ConvertFrom is expected to modify its receiver to contain the converted object.
Most of the conversion is straightforward copying, except for converting our changed field.
*/
// ConvertFrom converts from the Hub version (v1beta2) to this version.
func (dst *Builder) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1beta2.Builder)
	dst.ObjectMeta = src.ObjectMeta

	dst.Spec.Builder = src.Spec.Builder
	dst.Spec.Dockerfile = src.Spec.Dockerfile
	dst.Spec.Timeout = src.Spec.Timeout
	dst.Spec.Image = src.Spec.Image
	dst.Spec.ImageCredentials = src.Spec.ImageCredentials
	dst.Spec.BuilderCredentials = src.Spec.BuilderCredentials
	dst.Spec.Env = src.Spec.Env
	dst.Spec.State = BuilderState(src.Spec.State)

	if src.Spec.SrcRepo != nil {
		dst.Spec.SrcRepo = &GitRepo{
			Url:           src.Spec.SrcRepo.Url,
			Revision:      src.Spec.SrcRepo.Revision,
			SourceSubPath: src.Spec.SrcRepo.SourceSubPath,
			Credentials:   src.Spec.SrcRepo.Credentials,
		}

		if src.Spec.SrcRepo.BundleContainer != nil {
			dst.Spec.SrcRepo.BundleContainer = &BundleContainer{
				Image: src.Spec.SrcRepo.BundleContainer.Image,
			}
		}
	}

	if src.Spec.Shipwright != nil {
		dst.Spec.Shipwright = &ShipwrightEngine{
			Timeout: src.Spec.Shipwright.Timeout,
		}

		if src.Spec.Shipwright.Strategy != nil {
			dst.Spec.Shipwright.Strategy = &Strategy{
				Name: src.Spec.Shipwright.Strategy.Name,
				Kind: src.Spec.Shipwright.Strategy.Kind,
			}
		}

		if src.Spec.Shipwright.Params != nil {
			dst.Spec.Params = make(map[string]string)
			for _, item := range src.Spec.Shipwright.Params {
				if item.SingleValue == nil || item.SingleValue.Value == nil {
					continue
				}

				dst.Spec.Params[item.Name] = *item.Value
			}
		}
	}

	// Status
	dst.Status.State = src.Status.State
	dst.Status.Phase = src.Status.Phase
	dst.Status.ResourceRef = src.Status.ResourceRef
	dst.Status.Reason = src.Status.Reason

	if src.Status.Output != nil {
		dst.Status.Output = &Output{
			Digest: src.Status.Output.Digest,
			Size:   src.Status.Output.Size,
		}
	}

	if src.Status.Sources != nil {
		for _, item := range src.Status.Sources {
			res := SourceResult{
				Name: item.Name,
			}

			if item.Git != nil {
				res.Git = &GitSourceResult{
					CommitSha:    item.Git.CommitSha,
					CommitAuthor: item.Git.CommitAuthor,
					BranchName:   item.Git.BranchName,
				}
			}

			if item.Bundle != nil {
				res.Bundle = &BundleSourceResult{
					Digest: item.Bundle.Digest,
				}
			}

			dst.Status.Sources = append(dst.Status.Sources, res)
		}
	}

	return nil
}
