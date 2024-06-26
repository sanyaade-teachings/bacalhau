"""Submit a job to the server."""

import json

from bacalhau_apiclient.api import job_api
from bacalhau_apiclient.models.legacy_cancel_request import LegacyCancelRequest
from bacalhau_apiclient.models.legacy_events_request import LegacyEventsRequest
from bacalhau_apiclient.models.legacy_list_request import LegacyListRequest
from bacalhau_apiclient.models.legacy_state_request import LegacyStateRequest
from bacalhau_apiclient.models.legacy_submit_request import LegacySubmitRequest
from bacalhau_apiclient.rest import ApiException
from bacalhau_sdk.config import (
    get_client_id,
    get_client_public_key,
    init_config,
    sign_for_client,
)

conf = init_config()
client = job_api.ApiClient(conf)
api_instance = job_api.JobApi(client)


def submit(data: dict):
    """Submit a job to the server.

    Input `data` object is sanittized and signed before being sent to the server.
    """
    sanitized_data = client.sanitize_for_serialization(data)
    json_data = json.dumps(sanitized_data, indent=None, separators=(", ", ": "))
    json_bytes = json_data.encode("utf-8")
    signature = sign_for_client(json_bytes)
    client_public_key = get_client_public_key()
    submit_req = LegacySubmitRequest(
        client_public_key=client_public_key,
        payload=sanitized_data,
        signature=signature,
    )
    return api_instance.submit(submit_req)


def cancel(job_id: str):
    """Cancels a job on the server."""
    payload = dict(
        ClientID=get_client_id(),
        JobID=job_id,
    )

    sanitized_data = client.sanitize_for_serialization(payload)
    json_data = json.dumps(payload, indent=None, separators=(", ", ": "))
    json_bytes = json_data.encode("utf-8")
    signature = sign_for_client(json_bytes)
    client_public_key = get_client_public_key()
    cancel_req = LegacyCancelRequest(
        client_public_key=client_public_key,
        payload=sanitized_data,
        signature=signature,
    )
    return api_instance.cancel(cancel_req)


def list():
    """List all jobs."""
    try:
        # Simply lists jobs.
        list_request = LegacyListRequest(
            client_id=get_client_id(),
            sort_reverse=False,
            sort_by="created_at",
            return_all=False,
            max_jobs=5,
            exclude_tags=[],
            include_tags=[],
        )
        api_response = api_instance.list(list_request)
    except ApiException as e:
        print("Exception when calling JobApi->list: %s\n" % e)
    return api_response


def results(job_id: str):
    """Get results."""
    try:
        # Returns the results of the job-id specified in the body payload.
        state_request = StateRequest(
            client_id=get_client_id(),
            job_id=job_id,
        )
        api_response = api_instance.results(state_request)
    except ApiException as e:
        print("Exception when calling JobApi->results: %s\n" % e)
    return api_response


def states(job_id: str):
    """Get states."""
    try:
        # Returns the state of the job-id specified in the body payload.
        state_request = LegacyStateRequest(
            client_id=get_client_id(),
            job_id=job_id,
        )
        api_response = api_instance.states(state_request)
    except ApiException as e:
        print("Exception when calling JobApi->states: %s\n" % e)
    return api_response


def events(job_id: str):
    """Get events."""
    # TODO - add tests
    try:
        # Returns the events of the job-id specified in the body payload.
        state_request = LegacyEventsRequest(
            client_id=get_client_id(),
            job_id=job_id,
        )
        api_response = api_instance.events(state_request)
    except ApiException as e:
        print("Exception when calling JobApi->events: %s\n" % e)
    return api_response
