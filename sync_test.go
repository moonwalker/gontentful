package gontentful

const publishEntry = `
{
    "sys": {
        "type": "Array"
    },
    "items": [
        {
            "sys": {
                "space": {
                    "sys": {
                        "type": "Link",
                        "linkType": "Space",
                        "id": "dbq0oal15rwl"
                    }
                },
                "id": "7qpbdz6NUtWA6kS72QJeuB",
                "type": "Entry",
                "createdAt": "2019-03-22T02:06:40.875Z",
                "updatedAt": "2019-03-30T14:22:39.139Z",
                "environment": {
                    "sys": {
                        "id": "master",
                        "type": "Link",
                        "linkType": "Environment"
                    }
                },
                "revision": 2,
                "contentType": {
                    "sys": {
                        "type": "Link",
                        "linkType": "ContentType",
                        "id": "article"
                    }
                }
            },
            "fields": {
                "slug": {
                    "en": "kptest"
                },
                "title": {
                    "en": "kptest",
                    "en-SE": "kptest_en-se"
                },
                "content": {
                    "en": "kptest content sa\n"
				},
				"backgroundImage": {
                    "en": {
                        "sys": {
                            "type": "Link",
                            "linkType": "Entry",
                            "id": "355FyGlXxKoSOmasQgAysa"
                        }
                    }
                }
            }
		},
		{
			"sys": {
				"space": {
					"sys": {
						"type": "Link",
						"linkType": "Space",
						"id": "dbq0oal15rwl"
					}
				},
				"id": "1jwUI9Z4BgWOS4CKy4a4uK",
				"type": "Entry",
				"createdAt": "2018-03-27T11:48:21.542Z",
				"updatedAt": "2019-03-30T14:52:21.555Z",
				"environment": {
					"sys": {
						"id": "master",
						"type": "Link",
						"linkType": "Environment"
					}
				},
				"revision": 12,
				"contentType": {
					"sys": {
						"type": "Link",
						"linkType": "ContentType",
						"id": "product"
					}
				}
			},
			"fields": {
				"name": {
					"en": "dreamz"
				},
				"type": {
					"en": "casino"
				},
				"markets": {
					"en": [
						{
							"sys": {
								"type": "Link",
								"linkType": "Entry",
								"id": "2riJx4PYhe6io0u8ycMsiC"
							}
						},
						{
							"sys": {
								"type": "Link",
								"linkType": "Entry",
								"id": "6DLWPqwxq0w4EqMOAI6E04"
							}
						},
						{
							"sys": {
								"type": "Link",
								"linkType": "Entry",
								"id": "1M0ki6hpBqoQm26Mc8aw6m"
							}
						},
						{
							"sys": {
								"type": "Link",
								"linkType": "Entry",
								"id": "1QnuR9TiRK8YW2OOMWyWOA"
							}
						},
						{
							"sys": {
								"type": "Link",
								"linkType": "Entry",
								"id": "2GEQZiRNOMeC6MYiAYwca8"
							}
						},
						{
							"sys": {
								"type": "Link",
								"linkType": "Entry",
								"id": "562WWL4h7q226Swwskciqi"
							}
						},
						{
							"sys": {
								"type": "Link",
								"linkType": "Entry",
								"id": "3MdX8Nxp8QsMY8OQMcWsWG"
							}
						},
						{
							"sys": {
								"type": "Link",
								"linkType": "Entry",
								"id": "3QdbpFbes0QWiC220cUOQg"
							}
						},
						{
							"sys": {
								"type": "Link",
								"linkType": "Entry",
								"id": "6zbClN1OjmYYUiiG6k4qOQ"
							}
						}
					]
				},
				"defaultLocale": {
					"en": {
						"sys": {
							"type": "Link",
							"linkType": "Entry",
							"id": "2JbSbu7w2s6oAmwaAkQaqu"
						}
					}
				},
				"baseCurrency": {
					"en": {
						"sys": {
							"type": "Link",
							"linkType": "Entry",
							"id": "44JAeD9p28s8kYwaasU28G"
						}
					}
				},
				"showMarketSelector": {
					"en": true
				},
				"apiKeys": {
					"en": {
						"olark": "2462-416-10-4830",
						"segment": {
							"sources": {
								"web": {
									"sourceId": "fQANQAOd5N",
									"writeKey": "vRky9oDzaPXDh4jsXsfvRN43NX95VfAI"
								}
							}
						},
						"tagmanager": "GTM-PQ7R5BF",
						"webfontConfig": {
							"google": {
								"families": [
									"Material+Icons"
								]
							}
						}
					}
				}
			}
		}
    ],
    "nextSyncUrl": "http://cdn.contentful.com/spaces/dbq0oal15rwl/environments/master/sync?sync_token=w5ZGw6JFwqZmVcKsE8Kow4grw45QdyZIGzAkwoDDrCVxHFjCmMKVw5Zew7h7B3sWw5LChsKfOsK1w50ERwTDuTx_woMOwpPClsOdw4YLwoDDniU9CXE_wpLCrwkFRcOjBsOBOSTCp8KUDsK4wp5Gw4VFTA"
}`

const unpublishEntry = `
{
    "sys": {
        "type": "Array"
    },
    "items": [
        {
            "sys": {
                "type": "DeletedEntry",
                "id": "7qpbdz6NUtWA6kS72QJeuB",
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
                "revision": 2,
                "createdAt": "2019-03-30T14:24:21.037Z",
                "updatedAt": "2019-03-30T14:24:21.037Z",
                "deletedAt": "2019-03-30T14:24:21.037Z"
            }
        }
    ],
    "nextSyncUrl": "http://cdn.contentful.com/spaces/dbq0oal15rwl/environments/master/sync?sync_token=w5ZGw6JFwqZmVcKsE8Kow4grw45QdyY4Lyk1w4hlHcOywoxkwpPDqsKeah9rAcOzQFpqU2zCllXCjMOewoVqwpwewqTCocK7wo3DosObNMOkw4fDnzAzG8KUSUx7wok7Z8OqdGg3w6DDlMKOw4PCtgHClMKgwqE"
}`

const publishAsset = `
{
    "sys": {
        "type": "Array"
    },
    "items": [
        {
            "sys": {
                "space": {
                    "sys": {
                        "type": "Link",
                        "linkType": "Space",
                        "id": "dbq0oal15rwl"
                    }
                },
                "id": "3KKHzDRm3C6QekKoOGWGea",
                "type": "Asset",
                "createdAt": "2018-09-13T15:20:01.840Z",
                "updatedAt": "2019-03-30T14:31:29.170Z",
                "environment": {
                    "sys": {
                        "id": "master",
                        "type": "Link",
                        "linkType": "Environment"
                    }
                },
                "revision": 4
            },
            "fields": {
                "title": {
                    "en": "bg-650-noiseless"
                },
                "file": {
                    "en": {
                        "url": "//images.ctfassets.net/dbq0oal15rwl/3KKHzDRm3C6QekKoOGWGea/30798260afacc1f1d874c7291e2a9216/bg-650-noiseless.jpg",
                        "details": {
                            "size": 100312,
                            "image": {
                                "width": 650,
                                "height": 1268
                            }
                        },
                        "fileName": "bg-650-noiseless.jpg",
                        "contentType": "image/jpeg"
                    }
                }
            }
        }
    ],
    "nextSyncUrl": "http://cdn.contentful.com/spaces/dbq0oal15rwl/environments/master/sync?sync_token=w5ZGw6JFwqZmVcKsE8Kow4grw45QdyY7w6Ipc8OfwpXCtsOmwr8nBMKrEsOmwr1_w4FYw5rCvhA4w4LDosONTXfDnA_Di8KkX8OcZsOLT8OewpZ2wrDDghzDucK9w7HCuXLDjAxzeVVZwojCt30fYcOnOVvDk8Klwqs"
}`
