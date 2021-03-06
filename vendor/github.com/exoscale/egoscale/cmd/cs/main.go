package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/exoscale/egoscale"
	"github.com/go-ini/ini"
	"github.com/urfave/cli"
)

const userDocumentationURL = "http://cloudstack.apache.org/api/apidocs-4.4/user/%s.html"
const rootDocumentationURL = "http://cloudstack.apache.org/api/apidocs-4.4/root_admin/%s.html"

var _client = new(egoscale.Client)

func main() {
	// global flags
	var debug bool
	var dryRun bool
	var dryJSON bool
	var region string
	var theme string
	var innerDebug bool
	var innerRegion string
	var innerDryRun bool

	// no prefix
	log.SetFlags(0)

	app := cli.NewApp()
	app.Name = "cs"
	app.HelpName = "cs"
	app.Usage = "CloudStack at the fingerprints"
	app.Description = "Exoscale Go CloudStack cli"
	app.Version = egoscale.Version
	app.Compiled = time.Now()
	app.EnableBashCompletion = true
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:        "debug, d",
			Usage:       "debug mode on",
			Destination: &debug,
		},
		cli.BoolFlag{
			Name:        "dry-run, D",
			Usage:       "produce a cURL ready URL",
			Destination: &dryRun,
			Hidden:      true,
		},
		cli.BoolFlag{
			Name:        "dry-json, j",
			Usage:       "produce a JSON preview of the query",
			Destination: &dryJSON,
			Hidden:      true,
		},
		cli.StringFlag{
			Name:        "region, r",
			Usage:       "cloudstack.ini file section name",
			Destination: &region,
		},
		cli.StringFlag{
			Name:        "theme, t",
			Usage:       "syntax highlighting theme, see: https://xyproto.github.io/splash/docs/",
			Value:       "",
			Destination: &theme,
		},
	}

	log.SetFlags(0)

	var method egoscale.Command
	app.Commands = buildCommands(&method, methods)
	for i, cmd := range app.Commands {
		// global, hidden debug flag
		cmd.Flags = append(cmd.Flags, cli.BoolFlag{
			Name:        "debug, d",
			Destination: &innerDebug,
			Hidden:      true,
		})
		cmd.Flags = append(cmd.Flags, cli.BoolFlag{
			Name:        "dry-run, D",
			Destination: &innerDryRun,
			Hidden:      true,
		})
		// global, hidden region flag
		cmd.Flags = append(cmd.Flags, cli.StringFlag{
			Name:        "region, r",
			Destination: &innerRegion,
			Hidden:      true,
		})

		app.Commands[i].Flags = cmd.Flags
	}

	app.Commands = append(app.Commands, cli.Command{
		Name:     "gen-doc",
		Hidden:   true,
		HideHelp: true,
		Action: func(c *cli.Context) error {
			generateDocs(app, "../../website/content/cs")
			return nil
		},
	})

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

	if method == nil {
		os.Exit(0)
	}

	// Picking a region
	if region == "" {
		if innerRegion == "" {
			r, ok := os.LookupEnv("CLOUDSTACK_REGION")
			if ok {
				region = r
			}
		} else {
			region = innerRegion
		}
	}

	client, _ := buildClient(region)
	if theme != "" {
		client.Theme = theme
	}

	// Show request and quit
	if debug || innerDebug {
		payload, errP := client.Payload(method)
		if errP != nil {
			if _, err := fmt.Fprintf(os.Stderr, "cannot build payload: %s\n", errP); err != nil {
				log.Fatal(err)
			}
		} else {
			qs := payload.Encode()
			fmt.Printf("%s\\\n?%s\n", client.Endpoint, strings.Replace(qs, "&", "\\\n&", -1))
		}

		apiName := client.APIName(method)
		resp, err := client.Request(&egoscale.ListAPIs{Name: apiName})
		if err != nil {
			log.Fatal(err)
		}

		help, err := formatHelp(resp.(*egoscale.ListAPIsResponse).API[0])
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println()
		printJSON(string(help), client.Theme)

		os.Exit(0)
	}

	if dryRun || innerDryRun {
		payload, err := client.Payload(method)
		if err != nil {
			log.Fatal(err)
		}
		signature, err := client.Sign(payload)
		if err != nil {
			log.Fatal(err)
		}

		payload.Add("signature", signature)

		if _, err := fmt.Fprintf(os.Stdout, "%s?%s\n", client.Endpoint, payload.Encode()); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}

	if dryJSON {
		request, err := json.MarshalIndent(method, "", "  ")
		if err != nil {
			log.Panic(err)
		}

		printJSON(string(request), client.Theme)
		os.Exit(0)
	}

	resp, err := client.Request(method)
	if err != nil {
		log.Fatal(err)
	}
	out, _ := json.MarshalIndent(&resp, "", "  ")
	printJSON(string(out), client.Theme)
}

func buildClient(region string) (*Client, error) {
	usr, _ := user.Current()
	localConfig, _ := filepath.Abs("cloudstack.ini")
	inis := []string{
		localConfig,
		filepath.Join(usr.HomeDir, ".cloudstack.ini"),
	}
	config := ""
	for _, i := range inis {
		if _, err := os.Stat(i); err != nil {
			continue
		}
		config = i
		break
	}

	if config == "" {
		log.Fatalf("Config file not found within: %s", strings.Join(inis, ", "))
	}

	cfg, err := ini.LoadSources(ini.LoadOptions{IgnoreInlineComment: true}, config)
	if err != nil {
		log.Fatal(err)
	}

	if region == "" {
		region = "cloudstack"
	}

	section, err := cfg.GetSection(region)
	if err != nil {
		log.Fatalf("Section %q not found in the config file %s", region, config)
	}
	endpoint := "https://api.exoscale.ch/compute"
	ep, err := section.GetKey("endpoint")
	if err == nil {
		endpoint = ep.String()
	}

	key, errKey := section.GetKey("key")
	secret, errSecret := section.GetKey("secret")

	if errKey != nil || errSecret != nil {
		log.Fatalf("Section %q is missing key or secret", region)
	}

	cs := egoscale.NewClient(endpoint, key.String(), secret.String())

	client := &Client{cs, ""}

	th, err := section.GetKey("theme")
	if err == nil {
		client.Theme = th.String()
	} else {
		section, err = cfg.GetSection("exoscale")
		if err == nil {
			theme, _ := section.GetKey("theme")
			client.Theme = theme.String()
		}
	}

	return client, nil
}

func buildCommands(out *egoscale.Command, methods map[string][]cmd) []cli.Command {
	commands := make([]cli.Command, 0)

	for category, ms := range methods {
		for i := range ms {
			s := ms[i]

			name := _client.APIName(s.command)
			description := _client.APIDescription(s.command)

			url := userDocumentationURL
			if s.hidden {
				url = rootDocumentationURL
			}

			cmd := cli.Command{
				Name:        name,
				Description: fmt.Sprintf("%s <%s>", description, fmt.Sprintf(url, name)),
				Category:    category,
				HideHelp:    s.hidden,
				Hidden:      s.hidden,
				Flags:       buildFlags(s.command),
			}
			// report back the current command
			cmd.Action = func(c *cli.Context) error {
				if len(c.Args()) > 0 {
					return fmt.Errorf("unexpected extra arguments found: %s", strings.Join(c.Args(), " "))
				}
				*out = s.command
				return nil
			}
			// bash autocomplete
			cmd.BashComplete = func(c *cli.Context) {
				val := reflect.ValueOf(s.command)
				// we've got a pointer
				value := val.Elem()

				if value.Kind() != reflect.Struct {
					log.Fatalf("struct was expected")
				}

				ty := value.Type()
				for i := 0; i < value.NumField(); i++ {
					field := ty.Field(i)

					argName := ""
					if json, ok := field.Tag.Lookup("json"); ok {
						tags := strings.Split(json, ",")
						argName = tags[0]
					}

					if argName == "" {
						continue
					}

					if !c.IsSet(argName) {
						fmt.Printf("--%s\n", argName)
					}
				}
			}
			commands = append(commands, cmd)
		}
	}

	return commands
}

func buildFlags(method egoscale.Command) []cli.Flag {
	flags := make([]cli.Flag, 0)

	val := reflect.ValueOf(method)
	// we've got a pointer
	value := val.Elem()

	if value.Kind() != reflect.Struct {
		log.Fatalf("struct was expected")
		return flags
	}

	ty := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := ty.Field(i)

		if field.Name == "_" {
			continue
		}

		// XXX refactor with request.go
		var argName string
		required := false
		if json, ok := field.Tag.Lookup("json"); ok {
			tags := strings.Split(json, ",")
			argName = tags[0]
			required = true
			for _, tag := range tags {
				if tag == "omitempty" {
					required = false
				}
			}
			if argName == "" || argName == "omitempty" {
				continue
			}
		}

		description := ""
		if required {
			description = "required"
		}

		if doc, ok := field.Tag.Lookup("doc"); ok {
			if description != "" {
				description = fmt.Sprintf("[%s] %s", description, doc)
			} else {
				description = doc
			}
		}

		val := value.Field(i)
		addr := val.Addr().Interface()
		switch val.Kind() {
		case reflect.Bool:
			flags = append(flags, cli.BoolFlag{
				Name:        argName,
				Usage:       description,
				Destination: addr.(*bool),
			})
		case reflect.Int:
			flags = append(flags, cli.IntFlag{
				Name:        argName,
				Usage:       description,
				Destination: addr.(*int),
			})
		case reflect.Int64:
			flags = append(flags, cli.Int64Flag{
				Name:        argName,
				Usage:       description,
				Destination: addr.(*int64),
			})
		case reflect.Uint:
			flags = append(flags, cli.UintFlag{
				Name:        argName,
				Usage:       description,
				Destination: addr.(*uint),
			})
		case reflect.Uint64:
			flags = append(flags, cli.Uint64Flag{
				Name:        argName,
				Usage:       description,
				Destination: addr.(*uint64),
			})
		case reflect.Float64:
			flags = append(flags, cli.Float64Flag{
				Name:        argName,
				Usage:       description,
				Destination: addr.(*float64),
			})
		case reflect.Int16:
			flag := cli.GenericFlag{
				Name:  argName,
				Usage: description,
			}
			typeName := field.Type.Name()
			if typeName != "int16" {
				flag.Value = &intTypeGeneric{
					addr:    addr,
					base:    10,
					bitSize: 16,
					typ:     field.Type,
				}
			} else {
				flag.Value = &int16Generic{
					value: addr.(*int16),
				}
			}
			flags = append(flags, flag)
		case reflect.Uint8:
			flags = append(flags, cli.GenericFlag{
				Name:  argName,
				Usage: description,
				Value: &uint8Generic{
					value: addr.(*uint8),
				},
			})
		case reflect.Uint16:
			flags = append(flags, cli.GenericFlag{
				Name:  argName,
				Usage: description,
				Value: &uint16Generic{
					value: addr.(*uint16),
				},
			})
		case reflect.String:
			typeName := field.Type.Name()
			if typeName != "string" {
				flags = append(flags, cli.GenericFlag{
					Name:  argName,
					Usage: description,
					Value: &stringerTypeGeneric{
						addr: addr,
						typ:  field.Type,
					},
				})
			} else {
				flags = append(flags, cli.StringFlag{
					Name:        argName,
					Usage:       description,
					Destination: addr.(*string),
				})
			}
		case reflect.Slice:
			switch field.Type.Elem().Kind() {
			case reflect.Uint8:
				ip := addr.(*net.IP)
				if *ip == nil || (*ip).Equal(net.IPv4zero) || (*ip).Equal(net.IPv6zero) {
					flags = append(flags, cli.GenericFlag{
						Name:  argName,
						Usage: description,
						Value: &ipGeneric{
							value: ip,
						},
					})
				}
			case reflect.String:
				flags = append(flags, cli.StringSliceFlag{
					Name:  argName,
					Usage: description,
					Value: (*cli.StringSlice)(addr.(*[]string)),
				})
			default:
				switch field.Type.Elem() {
				case reflect.TypeOf(egoscale.ResourceTag{}):
					flags = append(flags, cli.GenericFlag{
						Name:  argName,
						Usage: description,
						Value: &tagGeneric{
							value: addr.(*[]egoscale.ResourceTag),
						},
					})
				case reflect.TypeOf(egoscale.CIDR{}):
					flags = append(flags, cli.GenericFlag{
						Name:  argName,
						Usage: description,
						Value: &cidrListGeneric{
							value: addr.(*[]egoscale.CIDR),
						},
					})
				case reflect.TypeOf(egoscale.UUID{}):
					flags = append(flags, cli.GenericFlag{
						Name:  argName,
						Usage: description,
						Value: &uuidListGeneric{
							value: addr.(*[]egoscale.UUID),
						},
					})
				case reflect.TypeOf(egoscale.UserSecurityGroup{}):
					flags = append(flags, cli.GenericFlag{
						Name:  argName,
						Usage: description,
						Value: &userSecurityGroupListGeneric{
							value: addr.(*[]egoscale.UserSecurityGroup),
						},
					})
				default:
					//log.Printf("[SKIP] Slice of %s is not supported!", field.Name)
				}
			}
		case reflect.Map:
			key := reflect.TypeOf(val.Interface()).Key()
			switch key.Kind() {
			case reflect.String:
				flags = append(flags, cli.GenericFlag{
					Name:  argName,
					Usage: description,
					Value: &mapGeneric{
						value: addr.(*map[string]string),
					},
				})
			default:
				log.Printf("[SKIP] Type map for %s is not supported!", field.Name)
			}
		case reflect.Ptr:
			switch field.Type.Elem() {
			case reflect.TypeOf(true):
				flags = append(flags, cli.GenericFlag{
					Name:  argName,
					Usage: description,
					Value: &boolPtrGeneric{
						value: addr.(**bool),
					},
				})
			case reflect.TypeOf(egoscale.CIDR{}):
				flags = append(flags, cli.GenericFlag{
					Name:  argName,
					Usage: description,
					Value: &cidrGeneric{
						value: addr.(**egoscale.CIDR),
					},
				})
			case reflect.TypeOf(egoscale.UUID{}):
				flags = append(flags, cli.GenericFlag{
					Name:  argName,
					Usage: description,
					Value: &uuidGeneric{
						value: addr.(**egoscale.UUID),
					},
				})
			default:
				log.Printf("[SKIP] Ptr type of %s is not supported!", field.Name)
			}
		default:
			log.Printf("[SKIP] Type of %s is not supported! %v", field.Name, val.Kind())
		}
	}

	return flags
}

// Client holds the internal meta information for the cli
type Client struct {
	*egoscale.Client
	Theme string
}

func formatHelp(api egoscale.API) ([]byte, error) {
	response := make(map[string]interface{})
	gather(response, api.Response)

	return json.MarshalIndent(response, "", "  ")
}

func gather(r map[string]interface{}, fields []egoscale.APIField) {
	for _, field := range fields {
		if len(field.Response) > 0 {
			response := make(map[string]interface{})
			gather(response, field.Response)
			r[field.Name] = []map[string]interface{}{response}
		} else {
			r[field.Name] = fmt.Sprintf("%s (%s) - %s", field.Name, field.Type, field.Description)
		}
	}
}
