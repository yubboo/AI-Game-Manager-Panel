package dedicated

import (
	"errors"
	"fmt"
	"strings"
)

var ErrMalformedExtraArgs = errors.New("malformed extra launch arguments")

// BuildLaunchArgs preserves the Python baseline ordering:
//
//	[-conf_dir X] -cluster C -shard S [-ugc_directory X] [extra...]
func BuildLaunchArgs(clusterName, shardName, confDirArg, ugcDirectory, extraArgs string) ([]string, error) {
	if strings.TrimSpace(clusterName) == "" {
		return nil, errors.New("cluster name is required")
	}
	if strings.TrimSpace(shardName) == "" {
		return nil, errors.New("shard name is required")
	}
	args := []string{"-cluster", clusterName, "-shard", shardName}
	if strings.TrimSpace(ugcDirectory) != "" {
		args = append(args, "-ugc_directory", ugcDirectory)
	}
	if strings.TrimSpace(confDirArg) != "" {
		args = append([]string{"-conf_dir", confDirArg}, args...)
	}
	if strings.TrimSpace(extraArgs) != "" {
		parsed, err := splitWindowsStyleArgs(extraArgs)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrMalformedExtraArgs, err)
		}
		for _, item := range parsed {
			if strings.EqualFold(strings.TrimSpace(item), "-console") {
				continue
			}
			args = append(args, item)
		}
	}
	return args, nil
}

func BuildLaunchSpec(install Installation, documentsDir string, request LaunchRequest) (LaunchSpec, error) {
	if !install.Valid || install.Executable == "" || install.BinDir == "" {
		return LaunchSpec{}, errors.New("DST Dedicated Server installation is not valid")
	}
	confArg, err := ResolveConfDirArg(documentsDir, request.KleiRoot)
	if err != nil {
		return LaunchSpec{}, err
	}
	args, err := BuildLaunchArgs(request.ClusterName, request.ShardName, confArg, request.UGCDirectory, request.ExtraArgs)
	if err != nil {
		return LaunchSpec{}, err
	}
	role := ShardSecondary
	if strings.EqualFold(request.ShardName, string(ShardMaster)) {
		role = ShardMaster
	}
	return LaunchSpec{
		ClusterName:      request.ClusterName,
		ShardName:        request.ShardName,
		Role:             role,
		Executable:       install.Executable,
		WorkingDirectory: install.BinDir,
		Arguments:        args,
		ConfDirArgument:  confArg,
		UGCDirectory:     request.UGCDirectory,
		ExtraArgs:        request.ExtraArgs,
		Architecture:     install.Architecture,
	}, nil
}

// splitWindowsStyleArgs mirrors Python shlex.split(value, posix=False) used by
// DSTCamp. In non-POSIX mode a quote only starts quoted parsing when it is the
// first character of a token; a quote encountered after ordinary token text is
// treated as a normal character. DSTCamp then strips one matching quote pair
// only when the entire token is wrapped in that pair.
func splitWindowsStyleArgs(value string) ([]string, error) {
	runes := []rune(value)
	result := make([]string, 0)
	for i := 0; i < len(runes); {
		for i < len(runes) && isShlexWhitespace(runes[i]) {
			i++
		}
		if i >= len(runes) {
			break
		}

		start := i
		if runes[i] == '\'' || runes[i] == '"' {
			quote := runes[i]
			i++
			for i < len(runes) && runes[i] != quote {
				i++
			}
			if i >= len(runes) {
				return nil, errors.New("No closing quotation")
			}
			i++ // include the closing quote, exactly like shlex posix=False
		} else {
			for i < len(runes) && !isShlexWhitespace(runes[i]) {
				i++
			}
		}

		token := string(runes[start:i])
		if len(token) >= 2 {
			first, last := token[0], token[len(token)-1]
			if (first == '"' || first == '\'') && first == last {
				token = token[1 : len(token)-1]
			}
		}
		result = append(result, token)
	}
	return result, nil
}

func isShlexWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\r' || r == '\n'
}
