package main

import (
	"encoding/json"
	"log"

	"github.com/spf13/cobra"

	"github.com/moonwalker/gontentful"
)

const publishRequest = `{
	"sys": {
	  "type": "Entry",
	  "id": "4jUt2vZ6bklScusycZ7mrF",
	  "space": {
		"sys": {
		  "type": "Link",
		  "linkType": "Space",
		  "id": "dbq0oal15rwl"
		}
	  },
	  "environment": {
		"sys": {
		  "id": "master",
		  "type": "Link",
		  "linkType": "Environment"
		}
	  },
	  "contentType": {
		"sys": {
		  "type": "Link",
		  "linkType": "ContentType",
		  "id": "game"
		}
	  },
	  "revision": 5,
	  "createdAt": "2019-04-10T13:01:40.024Z",
	  "updatedAt": "2020-08-18T09:50:02.749Z"
	},
	"fields": {
	  "slug": {
		"en": "wild-worlds"
	  },
	  "provider": {
		"en": {
		  "sys": {
			"type": "Link",
			"linkType": "Entry",
			"id": "6hWTMXx9AsQIoEMGW20kQc"
		  }
		}
	  },
	  "studio": {
		"en": {
		  "sys": {
			"type": "Link",
			"linkType": "Entry",
			"id": "44oT5qYkta40wkqm00Ym60"
		  }
		}
	  },
	  "themes": {
		"en": [
		  {
			"sys": {
			  "type": "Link",
			  "linkType": "Entry",
			  "id": "1NKIDhxMYUiga2eSUe8Ec8"
			}
		  },
		  {
			"sys": {
			  "type": "Link",
			  "linkType": "Entry",
			  "id": "sUuf3bNmJUW6CsE0Auoia"
			}
		  }
		]
	  },
	  "winFeatures": {
		"en": [
		  {
			"sys": {
			  "type": "Link",
			  "linkType": "Entry",
			  "id": "wrhrqSWcuWgCMkgcKeG8q"
			}
		  },
		  {
			"sys": {
			  "type": "Link",
			  "linkType": "Entry",
			  "id": "2JrYbvrVfGe2wAumsUImQM"
			}
		  }
		]
	  },
	  "bonusFeatures": {
		"en": [
		  {
			"sys": {
			  "type": "Link",
			  "linkType": "Entry",
			  "id": "4jhwZ7dhOo6qgyugOm08ka"
			}
		  },
		  {
			"sys": {
			  "type": "Link",
			  "linkType": "Entry",
			  "id": "3ZhOyJ7r21KNLo2U7PDpYu"
			}
		  }
		]
	  },
	  "wildFeatures": {
		"en": [
		  {
			"sys": {
			  "type": "Link",
			  "linkType": "Entry",
			  "id": "63eG6k8A7egoYEIeMcSM6E"
			}
		  },
		  {
			"sys": {
			  "type": "Link",
			  "linkType": "Entry",
			  "id": "6g7iMllDFK6iosoqs2gQYQ"
			}
		  }
		]
	  },
	  "content": {
		"en": {
		  "sys": {
			"type": "Link",
			"linkType": "Entry",
			"id": "1wCCQxyXGIeE1KYOi8x1S4"
		  }
		}
	  },
	  "category": {
		"en": {
		  "sys": {
			"type": "Link",
			"linkType": "Entry",
			"id": "7t2g3lYNRmWiS6SckIMw8S"
		  }
		}
	  },
	  "format": {
		"en": "standard"
	  },
	  "deviceConfigurations": {
		"en": [
		  {
			"sys": {
			  "type": "Link",
			  "linkType": "Entry",
			  "id": "7f1Fa0cgJt5ALmIOlA1eT3"
			}
		  },
		  {
			"sys": {
			  "type": "Link",
			  "linkType": "Entry",
			  "id": "EolTtkSur4rMFizshAZWp"
			}
		  }
		]
	  },
	  "type": {
		"en": {
		  "sys": {
			"type": "Link",
			"linkType": "Entry",
			"id": "5wfHEdAbyESaOMQYYwAqwS"
		  }
		}
	  },
	  "priority": {
		"en": 1855
	  }
	}
  }`

func init() {
	publishCmd.AddCommand(pgPublishCmd)
}

var pgPublishCmd = &cobra.Command{
	Use:   "pg",
	Short: "Publish content",

	Run: func(cmd *cobra.Command, args []string) {
		client := gontentful.NewClient(&gontentful.ClientOptions{
			CdnURL:        apiURL,
			SpaceID:       spaceID,
			EnvironmentID: environmentID,
			CdnToken:      cdnToken,
			CmaURL:        cmaURL,
			CmaToken:      cmaToken,
		})

		log.Println("get space...")
		space, err := client.Spaces.GetSpace()
		if err != nil {
			log.Fatal(err)
		}
		log.Println("get types...")
		types, err := client.ContentTypes.GetTypes()
		if err != nil {
			log.Fatal(err)
		}
		log.Println("get types done")

		item := &gontentful.PublishedEntry{}
		err = json.Unmarshal([]byte(publishRequest), item)
		if err != nil {
			log.Fatal(err)
		}

		var contentModel *gontentful.ContentType
		for _, ct := range types.Items {
			if ct.Sys.ID == item.Sys.ContentType.Sys.ID {
				contentModel = ct
				break
			}
		}
		if contentModel == nil {
			log.Fatal("contentModel not found")
		}
		log.Printf("publishing content...")
		pub := gontentful.NewPGPublish(schemaName, space.Locales, contentModel, item, "published")
		err = pub.Exec(databaseURL)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("content published successfully")
	},
}
