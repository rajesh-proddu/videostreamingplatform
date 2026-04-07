video streaming platform

Functional Requirements
1. upload videos with http2 streams support
2. download videos with http2 streams support 
3. recommendation engine based upon users watch history 

Non functional Requirements
1. Hardly 1K requests per sec
2. Availability over consitency 


Architecture 
1. Modular monolith service for following operations
    1.1 Meta data service (stores all videos meta data in mysql)
    1.2 Data service provides (stores videos info in s3)
    1.3 Search (search over meta data) 
2. Recomendation engine (soft real time)
    2.1 Publish watch history events to kafka
    2.2 Set up Spark ingestion workflow for reading from kafka and storing them in iceberg 
    2.3 Use LLMs for recommendation with RAG (fetch user watch history and provide it to LLM)  

tests
1. Write solid UTs for each function
2. CTs for each service
3. E2E scripts for running happy and resilency test cases
4. Execute sanity tests after each deployment
5. perf runs to scale up nodes 


ci/cd
1. Use kind based k8s deployment in local
2. Use AWS EKS in prod
3. Github actions for static analysis, tests execution, compile, build and archiving images
4. Use terraform as IaaS for deployment in both local and AWS 
5. Add support for rolling upgrades 


operations
1. Use cloudwatch for logs, metrics in aws
2. Use grafana, promethoeus for local cloud 
3. All services MUST adopt open telemetry format for emitting metrics, traces 


Security
1. IAM role based access to AWS services,data 
2. No authenitication on APIs to begin    

Deployments
1. Run dev env locally
2. Prod in AWS 


Future extensions
1. API gateway infront of all services
2. Cross AZ deployments 
3. CDN usage for cross region latency 