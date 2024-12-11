package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/user"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/net"
	"github.com/shirou/gopsutil/process"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	serverCmd = &cobra.Command{
		Use:   "server",
		Short: "Start an inet_peercred server",
		Long:  "Start an inet_peercred server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer()
		},
	}
)

func init() {
	var certPath string
	var keyPath string

	serverCmd.Flags().StringVar(&certPath, "cert", "", "Path to the server certificate")
	serverCmd.Flags().StringVar(&keyPath, "key", "", "Path to the server key")

	viper.BindPFlag("cert", serverCmd.Flags().Lookup("cert"))
	viper.BindPFlag("key", serverCmd.Flags().Lookup("key"))

	viper.SetDefault("cert", "")
	viper.SetDefault("key", "")

	rootCmd.AddCommand(serverCmd)
}

type V1Query struct {
	LocalAddr  net.Addr `json:"local_addr"`
	RemoteAddr net.Addr `json:"remote_addr"`
}

type RESF struct {
	Real       string `json:"real"`
	Effective  string `json:"effective"`
	Saved      string `json:"saved"`
	Filesystem string `json:"filesystem"`
}

type V1Response struct {
	User                RESF     `json:"user"`
	Groups              RESF     `json:"groups"`
	SupplementaryGroups []string `json:"supplementary_groups"`
}

func findConnection(LocalAddr net.Addr, RemoteAddr net.Addr) (*net.ConnectionStat, error) {
	connections, err := net.Connections("inet")
	if err != nil {
		return nil, err
	}

	for _, c := range connections {
		if c.Laddr == LocalAddr && c.Raddr == RemoteAddr {
			return &c, nil
		}
	}

	return nil, nil
}

func extractQuery(w http.ResponseWriter, r *http.Request) (*V1Query, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	var query V1Query
	err = json.Unmarshal(body, &query)
	if err != nil {
		return nil, err
	}

	return &query, nil
}

func uidsToStrings(uids []int32) ([]string, error) {
	var users []string

	for _, uid := range uids {
		user, err := user.LookupId(fmt.Sprint(uid))
		if err != nil {
			return nil, err
		}
		users = append(users, user.Username)
	}

	return users, nil
}

func gidsToStrings(gids []int32) ([]string, error) {
	var groups []string

	for _, gid := range gids {
		group, err := user.LookupGroupId(fmt.Sprint(gid))
		if err != nil {
			return nil, err
		}
		groups = append(groups, group.Name)
	}

	return groups, nil
}

func getUserRESF(p *process.Process) (RESF, error) {
	uids, err := p.Uids()
	if err != nil {
		return RESF{}, err
	}

	uidsNames, err := uidsToStrings(uids)
	if err != nil {
		return RESF{}, err
	}

	return RESF{
		Real:       uidsNames[0],
		Effective:  uidsNames[1],
		Saved:      uidsNames[2],
		Filesystem: uidsNames[3],
	}, nil
}

func getGroupRESF(p *process.Process) (RESF, error) {
	gids, err := p.Gids()
	if err != nil {
		return RESF{}, err
	}

	gidsNames, err := gidsToStrings(gids)
	if err != nil {
		return RESF{}, err
	}

	return RESF{
		Real:       gidsNames[0],
		Effective:  gidsNames[1],
		Saved:      gidsNames[2],
		Filesystem: gidsNames[3],
	}, nil

}

func getSupplementaryGroups(p *process.Process) ([]string, error) {
	supplementaryGroups, err := p.Groups()
	if err != nil {
		return []string{}, err
	}
	supplementaryGroupsNames, err := gidsToStrings(supplementaryGroups)
	if err != nil {
		return []string{}, err
	}

	return supplementaryGroupsNames, nil
}

func v1_query(w http.ResponseWriter, r *http.Request) {
	log := log.With().Str("client", r.RemoteAddr).Logger()

	client_port := strings.Split(r.RemoteAddr, ":")[1]
	client_port_int, err := strconv.Atoi(client_port)
	if err != nil {
		http.Error(w, "Invalid client port", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Cannot convert client port to int")
		return
	}
	if client_port_int > 1024 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		log.Warn().Msg("Client port not privileged")
		return
	}

	query, err := extractQuery(w, r)
	if err != nil {
		http.Error(w, "Invalid query.", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Cannot extract query")
		return
	}

	log = log.With().Interface("query", query).Logger()

	connection, err := findConnection(query.LocalAddr, query.RemoteAddr)
	if err != nil {
		http.Error(w, "An internal server error occured.", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Cannot get connections")
		return
	}
	if connection == nil {
		http.NotFound(w, r)
		http.Error(w, "Connection not found", http.StatusNotFound)
		log.Trace().Msg("Connection not found")
		return
	}
	log.Trace().Interface("connection", connection).Msg("Connection found")

	p, err := process.NewProcess(connection.Pid)
	if err != nil {
		http.Error(w, "An internal server error occured.", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Cannot get process")
		return
	}

	user, err := getUserRESF(p)
	if err != nil {
		http.Error(w, "An internal server error occured.", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Cannot get user")
		return
	}

	groups, err := getGroupRESF(p)
	if err != nil {
		http.Error(w, "An internal server error occured.", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Cannot get groups")
		return
	}

	supplementaryGroupsNames, err := getSupplementaryGroups(p)
	if err != nil {
		http.Error(w, "An internal server error occured.", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Cannot get supplementary groups")
		return
	}

	response := V1Response{
		User:                user,
		Groups:              groups,
		SupplementaryGroups: supplementaryGroupsNames,
	}

	log.Info().
		Interface("response", response).
		Msg("Query Result")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func runServer() error {
	certPath := viper.GetString("cert")
	keyPath := viper.GetString("key")

	log.Info().
		Str("cert", certPath).
		Str("key", keyPath).
		Msg("Starting server")

	if certPath == "" || keyPath == "" {
		log.Fatal().Msg("Server requires a certificate and key")
	}

	http.HandleFunc("/v1/query", v1_query)

	return http.ListenAndServeTLS(":411", certPath, keyPath, nil)
}
