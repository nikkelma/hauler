package store

import (
	"context"
	"net/http"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/spf13/cobra"

	"github.com/rancherfederal/hauler/pkg/log"
	"github.com/rancherfederal/hauler/pkg/store"
)

type CopyOpts struct {
	Username  string
	Password  string
	Insecure  bool
	PlainHTTP bool
}

func (o *CopyOpts) AddFlags(cmd *cobra.Command) {
	f := cmd.Flags()

	f.StringVarP(&o.Username, "username", "u", "", "Username when copying to an authenticated remote registry")
	f.StringVarP(&o.Password, "password", "p", "", "Password when copying to an authenticated remote registry")
	f.BoolVar(&o.Insecure, "insecure", false, "Allow insecure connections when copying to a remote registry")
	f.BoolVar(&o.PlainHTTP, "plain-http", false, "Allow plain http connections when copying to a remote registry")

	// TODO: Regex matching
}

func CopyCmd(ctx context.Context, o *CopyOpts, s *store.Store, registry string) error {
	l := log.FromContext(ctx)

	refOpts := []name.Option{
		name.WithDefaultRegistry(registry),
	}
	remoteOpts := []remote.Option{
		remote.WithAuthFromKeychain(authn.DefaultKeychain),
	}

	if o.Username != "" || o.Password != "" {
		basicAuth := &authn.Basic{
			Username: o.Username,
			Password: o.Password,
		}
		remoteOpts = append(remoteOpts, remote.WithAuth(basicAuth))
	}

	if o.Insecure {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.TLSClientConfig.InsecureSkipVerify = true

		remoteOpts = append(remoteOpts, remote.WithTransport(transport))
	}

	if o.PlainHTTP {
		refOpts = append(refOpts, name.Insecure)
	}

	s.Open()
	defer s.Close()

	refs, err := s.List(ctx)
	if err != nil {
		return err
	}

	for _, r := range refs {
		ref, err := name.ParseReference(r, name.WithDefaultRegistry(s.Registry()))
		if err != nil {
			return err
		}

		o, err := remote.Image(ref)
		if err != nil {
			return err
		}

		rref, err := name.ParseReference(r, refOpts...)
		if err != nil {
			return err
		}

		l.Infof("copying [%s] -> [%s]", ref.Name(), rref.Name())
		if err := remote.Write(rref, o, remoteOpts...); err != nil {
			return err
		}
	}

	return nil
}
