package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	filepath2 "path/filepath"
	"strconv"
	"strings"
)

type Project struct {
	Name string `json:"name"`
}

type Cluster struct {
	Name     string `json:"name"`
	Location string `json:"location"`
}

type NamespaceJson struct {
	Items []struct {
		Metadata struct {
			Name string `json:"name"`
		}
	}
}

type NamespacesList struct {
	Namespaces []Namespace `yaml:"Namespaces"`
}

type Namespace struct {
	Name           string `yaml:"Name"`
	ConnectionData []ConnectionData `yaml:"ConnectionData"`
}

type ConnectionData struct {
	ProjectName string `yaml:"ProjectName"`
	ClusterName string `yaml:"ClusterName"`
	Region      string `yaml:"Region"`
}





func (namespacesList *NamespacesList) AddNamespace(namespace Namespace) {

	exist, index := namespacesList.NamespaceExists(namespace.Name)

	if exist {
		connectionData := append(namespacesList.Namespaces[index].ConnectionData, namespace.ConnectionData[0])
		namespacesList.Namespaces[index].ConnectionData = connectionData
	} else {
		namespacesList.Namespaces = append(namespacesList.Namespaces, namespace)
	}
}

func (namespacesList *NamespacesList) NamespaceExists(item string) (bool, int) {

	for i, ns := range namespacesList.Namespaces {
		if ns.Name == strings.TrimRight(item, "\n") {
			return true, i
		}
	}
	return false, 0
}

func (namespacesList *NamespacesList) SaveToFile() {

	filepath := getConfigFilePath()
	file, err := yaml.Marshal(namespacesList)
	if err != nil {
		fmt.Printf("%s", err)
	}
	_ = ioutil.WriteFile(filepath, file, 0644)

}

func (namespacesList *NamespacesList) LoadFromFile() {

	path := getConfigFilePath()
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, namespacesList)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
}


func (namespacesList *NamespacesList) GetNamespace(nsName string) (exists bool, namespace Namespace) {
	exists, index := namespacesList.NamespaceExists(nsName)

	if exists {
		return true, namespacesList.Namespaces[index]
	}
	return false, Namespace{}
}

func (n *Namespace) SetNamespace(name, projectName, clusterName, region string) {
	n.Name = name
	cd := ConnectionData{ProjectName: projectName, ClusterName: clusterName, Region: region}
	n.ConnectionData = append(n.ConnectionData, cd)
}

func (n *Namespace) GetConnectionDataForCluster(clusterName string)(cd ConnectionData){
	for i, item := range n.ConnectionData{
		if strings.Contains(item.ClusterName, clusterName){
			return n.ConnectionData[i]
		}
	}
	return ConnectionData{}
}

func (n *Namespace) GetConnectionDataForProject(projectName string)(cd ConnectionData){
	for i, item := range n.ConnectionData{
		if strings.Contains(item.ProjectName, projectName){
			return n.ConnectionData[i]
		}
	}
	return ConnectionData{}
}


func (namespacesList *NamespacesList)ScanNamespaces(project string){

	var projects []string
	if len(project) == 0  {
		projects = GetProjectsList()
	} else {
		projects = []string{project}
	}
	for _, s := range projects{
		fmt.Printf("--> Procesing project: %s\n", s)
		ActivateProject(s)
		clusters, _:= GetClusters()

		for _, c := range clusters{
			fmt.Printf("\t--> Processing cluster %s:\n", c.Name)
			ActivateCluster(c.Name, c.Location)
			namespaces := GetNamespaces()

			fmt.Printf("\t\tNamespaces in this clusterer are:\n")
			for _, n := range namespaces.Items{
				var namespace Namespace
				namespace.SetNamespace(n.Metadata.Name, s, c.Name, c.Location)
				namespacesList.AddNamespace(namespace)
				fmt.Printf("\t\t- %s\n", n.Metadata.Name)
			}

		}
	}
}


func getConfigFilePath()(fp string){
	home, _ := homedir.Dir()
	path := filepath2.Join(home, ".lmi")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, 0755)
	}
	fp = filepath2.Join(home, ".lmi", "namespaces.yaml")
	return
}

func GetProjectsList() (projectsList []string) {
	var jsonProjects []Project
	out, err := exec.Command("gcloud", "config", "configurations", "list", "--format", "json").Output()
	if err != nil {
		fmt.Printf("%s", err)
	}
	err = json.Unmarshal(out, &jsonProjects)
	if err != nil {
		fmt.Printf("Error parsing JSON object: %s\n", err)
	}
	for _, s := range jsonProjects {
		projectsList = append(projectsList, s.Name)
	}
	return
}



func ActivateProject(projectName string) (ok bool) {
	_, err := exec.Command("gcloud", "config", "configurations", "activate", projectName).Output()
	if err != nil {
		//fmt.Printf("%s", err)
		//fmt.Printf("Will return false now")
		return false
	}
	return true
}

func GetClusters() (clusters []Cluster, ok bool) {
	out, err := exec.Command("gcloud", "container", "clusters", "list", "--format", "json").Output()
	if err != nil {
		fmt.Printf("%s", err)
		ok = false
	}
	err = json.Unmarshal(out, &clusters)
	if err != nil {
		fmt.Printf("Something went wrong while pulling clusters for that project.\n %s", err)
	}
	ok = true
	return
}

func ActivateCluster(clusterName, regionName string) {
	_, err := exec.Command("gcloud", "container", "clusters", "get-credentials", clusterName, "--region",
		regionName).Output()
	if err != nil {
		fmt.Printf("%s", err)
	}
}

func GetNamespaces() (namespaces NamespaceJson) {
	out, err := exec.Command("kubectl", "get", "ns", "--request-timeout", "10s", "-o", "json").Output()
	if err != nil {
		fmt.Printf("%s", err)
	}
	err = json.Unmarshal(out, &namespaces)
	if err != nil {
		fmt.Printf("%s", err)
	}
	return
}


func SetContext(namespace string) {
	_, err := exec.Command("kubectl", "config", "set-context", "--current", "--namespace",
		namespace).Output()
	if err != nil {
		fmt.Printf("%s", err)
	}
}

func ConnectToNamespace(cd ConnectionData, namespace string) {
	ActivateProject(cd.ProjectName)
	ActivateCluster(cd.ClusterName, cd.Region)
	SetContext(namespace)
	fmt.Printf("You are now conected to:\n- namespace: %s\n- cluster: %s\n- project: %s\n", namespace,
		cd.ClusterName, cd.ProjectName)
}

var namespace string
var cluster string
var project string
var scan bool
var debug = false





// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "lmi NAMESPACE_NAME",
	Short: "This app allows you to quickly connect to specified namespace",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Args: cobra.ExactArgs(1),
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		if namespace == "" && len(args) == 0 && !scan{
			cmd.Usage()
			os.Exit(1)
		}

		if scan {
			if project != ""{
				var ns NamespacesList
				ns.LoadFromFile()
				ns.ScanNamespaces(project)
				ns.SaveToFile()
			} else {
				var ns NamespacesList
				ns.ScanNamespaces("")
				ns.SaveToFile()
			}

		} else {
			if len(args) > 0 {
				namespace = args[0]
			}
			var namespacesList NamespacesList
			namespacesList.LoadFromFile()
			reader := bufio.NewReader(os.Stdin)
			exist, ns := namespacesList.GetNamespace(namespace)
			if !exist {
				fmt.Printf("%s doesn't exist in configuration file !\nPlease duble check namesace name or perform scan to update configuration file.\n", namespace)
				os.Exit(2)
			}
			if len(ns.ConnectionData) > 1 {
				if cluster != "" {
					cd := ns.GetConnectionDataForCluster(cluster)
					if len(cd.ClusterName) > 0 {
						ConnectToNamespace(cd, namespace)
					} else {
						fmt.Printf("Config file doesn't have configuration for %s namespcae in cluster %s.\nPlease duble check yur input or perform scan to update configuration file.\n", namespace, cluster)
						os.Exit(2)
					}

				} else if project != ""{
					cd := ns.GetConnectionDataForProject(project)
					if len(cd.ProjectName) > 0 {
						ConnectToNamespace(cd, namespace)
					} else {
						fmt.Printf("Config file doesn't have configuration for %s namespcae in project %s.\nPlease duble check yur input or perform scan to update configuration file.\n", namespace, project)
						os.Exit(2)
					}
					ConnectToNamespace(ns.GetConnectionDataForProject(project), namespace)
				} else {
					fmt.Println("Which cluster you want to use:")
					for i, item := range ns.ConnectionData {
						fmt.Printf("%d) cluster:\t%s\tproject:\t%s\n", i, item.ClusterName, item.ProjectName)
					}
					text, _ := reader.ReadString('\n')
					text = strings.Trim(text, "\n")
					index, err := strconv.Atoi(text)
					if err != nil {
						fmt.Printf("Please input valid number.\n")
						os.Exit(2)
					}
					if index > len(ns.ConnectionData){
						fmt.Printf("Please input valid number.\n")
						os.Exit(2)
					}

					ConnectToNamespace(ns.ConnectionData[index], namespace)

				}

			} else {

				ConnectToNamespace(ns.ConnectionData[0], namespace)
			}
		}



	},
}


// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {

	rootCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Name of the namespace you want to connect to.")
	rootCmd.Flags().StringVarP(&cluster, "cluster", "c", "", "Name of the cluster you want to connect to, required if given namespace exist in more than one cluster.")
	rootCmd.Flags().StringVarP(&project, "project", "p", "", "Name of the project you want to connect to, required if given namespace exist in more than one project or you want to run scan only against one project.")
	rootCmd.Flags().BoolVarP(&scan, "scan", "s", false, "Scanning all or only given project to retrieves and store in configuration file all existing namespaces.")
}

func main() {

	Execute()

}

