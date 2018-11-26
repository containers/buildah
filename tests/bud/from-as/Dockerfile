FROM centos:7 AS base
RUN touch /1
ENV LOCAL=/1
RUN find $LOCAL
    	
FROM base
RUN find $LOCAL
