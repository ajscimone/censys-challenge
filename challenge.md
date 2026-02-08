# Overview

As part of the interview process for the Senior Software Engineer role, we ask candidates to complete a take-home assignment. This exercise contains ambiguities and areas that are not precisely defined, which is intentional. We want to see what decisions you make, what areas you focus on, and what trade offs you make. We will talk through these decisions in the later technical conversation with team members.

The scope is intentionally wide, but you should target approximately 4 hours of development time on this exercise. If you run out of time, focus on code quality, solving the major problems (rate limiting and access control) as well as documentation that explains what you would do in the areas you did not get to implement.

## The Scenario

Your team is building a system where Organizations can create Collections (datasets or saved searches). Collections have three access levels:

    Private - Only the creator can access
    Organization - All members of the org can access  
    Shared - Accessible via a share token to anyone with the token

## The Problem

Last week, a user shared a collection link on social media. It went viral - 50,000 requests in 5 minutes. The service fell over. You're building v2 to handle this.
Conflicting Stakeholder Requirements:

    Marketing/Sales: Share links should work without authentication (frictionless)
    Security: Must track who accesses what and enable instant revocation
    Product: Collections can be updated/deleted - share links must respect this in real-time

Think about the tradeoffs required by these stakeholder requirements. The README is an ideal place to add your assumptions and any future changes you would recommend for this service if provided more time.

## Service requirements

Build a Go service to the following specification.

    Implement a GRPC service that can perform CRUD operations on Collections
        This implementation can be basic (even in-memory)
    Supports sharing access to a collection through the use of a token
    Supports our 3 levels of access
    Supports token revocation/deletion
    Solves the main problem statement
    Maintains an access count for each share token

### Data Model - storage

Store Collections and Tokens in a database of your choice. Explain why you made this choice and how it might differ for a production ready application.

### Collections and Tokens 

Implement support for Collections resources via a GRPC service.

    Stores collections
    Supports the creation and deletion of sharing tokens to allow access to a collection
    Collections must support the 3 tiers of privacy

### Traffic Spike Protections

Implement a protection mechanism to prevent service degradation when placed under similar load as given in the problem statement.

### Recommended gRPC Service Contract for your service

```
service CollectionService {
    rpc CreateCollection(CreateCollectionRequest) returns (Collection);
    rpc GetCollection(GetCollectionRequest) returns (Collection);
    rpc UpdateCollection(UpdateCollectionRequest) returns (Collection);
    rpc DeleteCollection(DeleteCollectionRequest) returns (google.protobuf.Empty);

    // Sharing operations
    rpc CreateShareToken(CreateShareTokenRequest) returns (ShareToken);
    rpc GetSharedCollection(GetSharedCollectionRequest) returns (Collection);
    rpc RevokeShareToken(RevokeShareTokenRequest) returns (google.protobuf.Empty);
}
```

### Docker Setup

We require that you: 

    Provide the service + DB
    Use docker to run the application
    Include database migrations
