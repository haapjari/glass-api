package goplg

// 	// Append the https:// and .git prefix and postfix the RepositoryUrl variables.
// 	for i := 0; i < len(unprocessedRepositories); i++ {
// 		var databaseEntry models.Repository
// 		g.DatabaseClient.Table("repositories").Where("repository_url = ?", unprocessedRepositories[i].RepositoryUrl).First(&databaseEntry)
// 		// If the field "OriginalCodebaseSize" is empty, but it has a name, that means
// 		// it exists in the database, but is not analyzed yet -> proceed.
// 		if databaseEntry.OriginalCodebaseSize == "" && databaseEntry.RepositoryName != "" {
// 			waitGroup.Add(1)
// 			semaphore <- 1
// 			go func(i int) {
// 				log.Println("Processing repository: " + unprocessedRepositories[i].RepositoryName)
// 				repositoryUrl := "https://" + unprocessedRepositories[i].RepositoryUrl + ".git"
//
// 				err := utils.Command("git", "clone", "--depth", "1", repositoryUrl, "tmp"+"/"+unprocessedRepositories[i].RepositoryName)
// 				if err != nil {
// 					fmt.Printf("error while cloning repository %s: %s, skipping...\n", repositoryUrl, err)
// 				}
//
// 				repositoryCodeLines, err := g.calculateCodeLines("tmp" + "/" + unprocessedRepositories[i].RepositoryName)
// 				if err != nil {
// 					fmt.Print(err.Error())
// 				}
//
// 				goModsMutex.Lock()
// 				goMod, err := parseGoMod(unprocessedRepositories[i].RepositoryName + "/" + "go.mod")
// 				if err != nil {
// 					fmt.Printf("error, while parsing the modfile of "+unprocessedRepositories[i].RepositoryName+": %s", err)
// 				}
// 				g.GoMods[unprocessedRepositories[i].RepositoryUrl] = goMod
// 				goModsMutex.Unlock()
//
// 				g.generateDependenciesMap(unprocessedRepositories[i])
//
// 				var repositoryStruct models.Repository
// 				if err := g.DatabaseClient.Where("repository_url = ?", unprocessedRepositories[i].RepositoryUrl).First(&repositoryStruct).Error; err != nil {
// 					utils.CheckErr(err)
// 				}
//
// 				repositoryStruct.OriginalCodebaseSize = strconv.Itoa(repositoryCodeLines)
// 				unprocessedRepositories[i] = repositoryStruct
//
// 				g.DatabaseClient.Model(&repositoryStruct).Updates(repositoryStruct)
//
// 				defer func() {
// 					waitGroup.Done()
// 					<-semaphore
// 				}()
// 			}(i)
// 		}
// 		waitGroup.Wait()
// 		g.pruneTemporaryFolder()
//
// 		for !(len(semaphore) == 0) {
// 			continue
// 		}
// 	}
//
// 	return unprocessedRepositories
// }
//
// // Loop through repositories, generate the dependency map from the go.mod files of the
// // repositories, download the dependencies to the local disk, calculate their sizes and
// // save the values to the database.
// func (g *GoPlugin) processLibraries(repositoriesWithoutLibrarySize []models.Repository) []models.Repository {
// 	libraries := g.DependencyMap
// 	var mutex sync.Mutex
// 	var producerWaitGroup sync.WaitGroup
// 	var consumerWaitGroup sync.WaitGroup
// 	semaphore := make(chan int, g.MaxRoutines)
// 	utils.CopyFile("go.mod", "go.mod.bak")
// 	utils.CopyFile("go.sum", "go.sum.bak")
// 	os.Setenv("GOPATH", utils.GetProcessDirPath())
// 	for i, r := range repositoriesWithoutLibrarySize {
// 		repositoryName := r.RepositoryName
// 		repositoryUrl := r.RepositoryUrl
// 		totalLibraryCodeLines := 0
// 		if r.LibraryCodebaseSize == "" {
// 			log.Println(r.RepositoryName + " processing " + strconv.Itoa(len(libraries[repositoryName])) + " libraries...")
//
// 			// Calculate the Cached Values.
// 			for _, libraryUrl := range libraries[repositoryName] {
// 				value, ok := g.LibraryCache[libraryUrl]
// 				if ok {
// 					totalLibraryCodeLines += value
// 				}
// 			}
//
// 			calculateJobs := make(chan int)
// 			done := make(chan bool)
//
// 			// Producer
// 			// If the producer starts to lag out with the routines, cap them to core count.
// 			go func() {
// 				for j, libraryUrl := range libraries[repositoryName] {
// 					mutex.Lock()
// 					_, ok := g.LibraryCache[libraryUrl]
// 					mutex.Unlock()
// 					utils.RemoveFiles("go.mod", "go.sum")
// 					utils.CopyFile("go.mod.bak", "go.mod")
// 					utils.CopyFile("go.sum.bak", "go.sum")
// 					producerWaitGroup.Add(1)
// 					semaphore <- 1
// 					go func(j int, libraryUrl string) {
// 						if !ok {
// 							err := utils.Command("go", "get", "-d", "-v", downloadableFormat(libraryUrl))
// 							if err != nil {
// 								fmt.Printf("error while processing library %s: %s, skipping...\n", libraryUrl, err)
// 							}
// 							calculateJobs <- j
// 						}
// 						defer func() {
// 							<-semaphore
// 							producerWaitGroup.Done()
// 						}()
// 					}(j, libraryUrl)
// 				}
// 				producerWaitGroup.Wait()
// 				done <- true
// 			}()
//
// 			// Consumer
// 			go func() {
// 				for jobIndex := range calculateJobs {
// 					consumerWaitGroup.Add(1)
// 					go func(jobIndex int) {
// 						mutex.Lock()
// 						libraryUrl := parseLibraryUrl(libraries[repositoryName][jobIndex])
// 						mutex.Unlock()
// 						libraryCodeLines, err := g.calculateCodeLines(utils.GetProcessDirPath() + "/" + "pkg/mod" + "/" + libraryUrl)
// 						if err != nil {
// 							fmt.Println("error, while calculating library code lines:", err.Error())
// 						}
// 						mutex.Lock()
// 						g.LibraryCache[libraries[repositoryName][jobIndex]] = libraryCodeLines
// 						totalLibraryCodeLines += libraryCodeLines
// 						mutex.Unlock()
// 						defer func() {
// 							consumerWaitGroup.Done()
// 						}()
// 					}(jobIndex)
// 				}
// 			}()
// 			producerWaitGroup.Wait()
// 			consumerWaitGroup.Wait()
// 			<-done
// 			close(calculateJobs)
//
// 			g.pruneTemporaryFolder()
//
// 			utils.RemoveFiles("go.mod", "go.sum")
// 			utils.CopyFile("go.mod.bak", "go.mod")
// 			utils.CopyFile("go.sum.bak", "go.sum")
//
// 			var repositoryStruct models.Repository
// 			if err := g.DatabaseClient.Where("repository_url = ?", repositoryUrl).First(&repositoryStruct).Error; err != nil {
// 				utils.CheckErr(err)
// 			}
//
// 			repositoryStruct.LibraryCodebaseSize = strconv.Itoa(totalLibraryCodeLines)
// 			g.DatabaseClient.Model(&repositoryStruct).Updates(repositoryStruct)
//
// 			repositoriesWithoutLibrarySize[i] = repositoryStruct
// 		}
// 	}
//
// 	os.Setenv("GOPATH", utils.GetGoPath())
//
// 	utils.RemoveFiles("go.mod", "go.sum")
// 	utils.CopyFile("go.mod.bak", "go.mod")
// 	utils.CopyFile("go.sum.bak", "go.sum")
// 	utils.RemoveFiles("go.mod.bak", "go.sum.bak")
//
// 	return repositoriesWithoutLibrarySize
// }
//
// // Gets a list of repositories and returns a map of repository names and their dependencies,
// // which are parsed from the projects "go.mod" -file.
// func (g *GoPlugin) generateDependenciesMap(repository models.Repository) {
// 	repositoryName := repository.RepositoryName
// 	repositoryUrl := repository.RepositoryUrl
//
// 	g.DependencyMap[repositoryName] = append(g.DependencyMap[repositoryName], g.GoMods[repositoryUrl].Require...)
//
// 	if g.GoMods[repositoryUrl].Replace != nil {
// 		replacePaths := g.GoMods[repositoryUrl].Replace
// 		for i := 0; i < len(replacePaths); i++ {
// 			if isLocalPath(replacePaths[i]) {
// 				innerModFilePath := utils.GetProcessDirPath() + "/" + "pkg/mod" + "/" + repositoryUrl + trimFirstRune(replacePaths[i]) + "/" + "go.mod"
// 				innerModuleFile, err := parseGoMod(innerModFilePath)
// 				if err != nil {
// 					fmt.Printf("error, while parsing the inner module file: %s", err)
// 				} else {
// 					g.DependencyMap[repositoryName] = append(g.DependencyMap[repositoryName], innerModuleFile.Require...)
// 				}
// 			}
// 		}
// 	}
// 	g.DependencyMap[repositoryName] = utils.RemoveDuplicates(g.DependencyMap[repositoryName])
// }
